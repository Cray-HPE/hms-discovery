// MIT License
//
// (C) Copyright [2020-2022] Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	compcredentials "github.com/Cray-HPE/hms-compcredentials"
	"github.com/Cray-HPE/hms-discovery/internal/http_logger"
	"github.com/Cray-HPE/hms-discovery/pkg/pdu_credential_store"
	"github.com/Cray-HPE/hms-discovery/pkg/switches"
	dns_dhcp "github.com/Cray-HPE/hms-dns-dhcp/pkg"
	securestorage "github.com/Cray-HPE/hms-securestorage"
	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
	"github.com/Cray-HPE/hms-smd/pkg/sm"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/namsral/flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	slsURL = flag.String("sls_url", "http://cray-sls",
		"System Layout Service URL")
	hsmURL = flag.String("hsm_url", "http://cray-smd",
		"State Manager URL")
	capmcURL = flag.String("capmc_url", "http://cray-capmc",
		"CAPMC URL")

	discoverRiver = flag.Bool("discover_river", true, "Discover River nodes?")

	discoverMountain        = flag.Bool("discover_mountain", true, "Discover Mountain nodes?")
	mountainDiscoveryScript = flag.String("mountain_discovery_script", "mountain_discovery.py",
		"Location of the script to give Python to run for Mountain discovery.")

	discoverManagementVirtualNodes      = flag.Bool("discover_management_virtual_nodes", true, "Discover Management Virtual nodes")
	discoverManagementNodes             = flag.Bool("discover_management_nodes", true, "Discover Management Nodes")
	rediscoverFailedRedfishEndpoints    = flag.Bool("rediscover_failed_redfish_endpoints", true, "Rediscover Failed Redfish Endpoints")
	populateManagementSwitchCredentials = flag.Bool("populate_management_switch_credentials", true, "Populate management switch credentials")

	httpClient *retryablehttp.Client

	atomicLevel zap.AtomicLevel
	logger      *zap.Logger

	secureStorage       securestorage.SecureStorage
	hsmCredentialStore  *compcredentials.CompCredStore
	redsCredentialStore *switches.RedsCredStore
	pduCredentialStore  *pdu_credential_store.PDUCredentialStore

	dhcpdnsClient dns_dhcp.DNSDHCPHelper
)

type RedfishEndpointArray struct {
	RedfishEndpoints []rf.RedfishEPDescription `json:"RedfishEndpoints"`
}

func setupVault() error {
	var err error
	secureStorage, err = securestorage.NewVaultAdapter(os.Getenv("VAULT_BASE_PATH"))
	if err != nil {
		return err
	}

	hsmCredentialStore = compcredentials.NewCompCredStore("hms-creds", secureStorage)
	redsCredentialStore = switches.NewRedsCredStore("reds-creds", secureStorage)
	pduCredentialStore = pdu_credential_store.NewPDUCredStore("pdu-creds", secureStorage)

	return nil
}

func setupLogging() {
	logLevel := os.Getenv("LOG_LEVEL")
	logLevel = strings.ToUpper(logLevel)

	atomicLevel = zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atomicLevel,
	))

	switch logLevel {
	case "DEBUG":
		atomicLevel.SetLevel(zap.DebugLevel)
	case "INFO":
		atomicLevel.SetLevel(zap.InfoLevel)
	case "WARN":
		atomicLevel.SetLevel(zap.WarnLevel)
	case "ERROR":
		atomicLevel.SetLevel(zap.ErrorLevel)
	case "FATAL":
		atomicLevel.SetLevel(zap.FatalLevel)
	case "PANIC":
		atomicLevel.SetLevel(zap.PanicLevel)
	default:
		atomicLevel.SetLevel(zap.InfoLevel)
	}
}

func getNotDiscoveredOKEndpointFromHSM() (notDiscoveredEndpoints []rf.RedfishEPDescription) {
	url := fmt.Sprintf("%s/Inventory/RedfishEndpoints", *hsmURL)

	response, err := httpClient.Get(url)
	if err != nil {
		logger.Error("Failed to get RedfishEndpoints from HSM!", zap.Error(err))
		return
	}

	if response.StatusCode != http.StatusOK {
		logger.Error("Unexpected status code from HSM!", zap.Int("response.StatusCode", response.StatusCode))
		return
	}

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error("Failed to read body!", zap.Error(err))
		return
	}
	defer response.Body.Close()

	var allEndpoints RedfishEndpointArray
	err = json.Unmarshal(jsonBytes, &allEndpoints)
	if err != nil {
		logger.Error("Failed to unmarshal HSM Redfish endpoints JSON!", zap.Error(err))
		return
	}

	// Now we have all the endpoints, loop through them all looking only for those that are not DiscoveredOK
	// (or actively being discovered).
	for _, endpoint := range allEndpoints.RedfishEndpoints {
		if endpoint.DiscInfo.LastStatus != rf.DiscoverOK &&
			endpoint.DiscInfo.LastStatus != rf.DiscoveryStarted {
			logger.Debug("Found Redfish endpoint that was not DiscoveredOK/DiscoveryStarted.", zap.Any("endpoint", endpoint))

			notDiscoveredEndpoints = append(notDiscoveredEndpoints, endpoint)
		} else {
			logger.Debug("Found DiscoveredOK/DiscoveryStarted Redfish endpoint.", zap.Any("endpoint", endpoint))
		}
	}

	return
}

func reDiscoverEndpoints(endpoints []string) {
	// Cool, some endpoints appear reachable that aren't yet discovered, let's have HSM give them a go.
	discover := sm.DiscoverIn{
		XNames: endpoints,
		Force:  false,
	}

	payloadBytes, marshalErr := json.Marshal(discover)
	if marshalErr != nil {
		logger.Error("Failed to marshal HSM discover object!", zap.Error(marshalErr))
		return
	}

	hsmURL := fmt.Sprintf("%s/Inventory/Discover", *hsmURL)
	request, requestErr := retryablehttp.NewRequest("POST", hsmURL, bytes.NewBuffer(payloadBytes))
	if requestErr != nil {
		logger.Error("Failed to construct request!", zap.Error(requestErr))
		return
	}
	request.Header.Set("Content-Type", "application/json")

	response, doErr := httpClient.Do(request)
	if doErr != nil {
		logger.Error("Failed to execute POST request!", zap.Error(doErr))
		return
	}

	potentialLogger := logger.With(zap.Strings("potentiallyDiscoverableEndpoints", endpoints))
	if response.StatusCode == http.StatusOK {
		potentialLogger.Info("Told HSM to re-discover previously failed endpoints.")
	} else {
		potentialLogger.Error("Got unexpected status code from re-discover attempt from HSM!",
			zap.Int("response.StatusCode", response.StatusCode))
	}
}

func main() {
	// Parse the arguments.
	flag.Parse()

	*hsmURL = *hsmURL + "/hsm/v2"

	setupLogging()

	// For performance reasons we'll keep the client that was created for this base request and reuse it later.
	httpClient = retryablehttp.NewClient()
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient.HTTPClient.Transport = transport

	httpClient.RetryMax = 2
	httpClient.RetryWaitMax = time.Second * 2

	// Also, since we're using Zap logger it make sense to set the logger to use the one we've already setup.
	httpLogger := http_logger.NewHTTPLogger(logger)
	httpClient.Logger = httpLogger

	// Setup the DHCP/DNS client.
	dhcpdnsClient = dns_dhcp.NewDHCPDNSHelper(*hsmURL, httpClient)

	logger.Info("Beginning HMS discovery process.",
		zap.String("slsURL", *slsURL),
		zap.String("hsmURL", *hsmURL),
		zap.Bool("discoverRiver", *discoverRiver),
		zap.Bool("discoverMountain", *discoverMountain),
		zap.Bool("discoverManagementVirtualNodes", *discoverManagementVirtualNodes),
		zap.Bool("discoverManagementNodes", *discoverManagementNodes),
		zap.Bool("managementSwitchCredentials", *populateManagementSwitchCredentials),
		zap.Bool("rediscoverFailedRedfishEndpoints", *rediscoverFailedRedfishEndpoints),
		zap.String("atomicLevel", atomicLevel.String()),
	)

	// Loop waiting for the connection to Vault to work.
	var err error
	for {
		err = setupVault()
		if err != nil {
			logger.Error("Unable to setup Vault!", zap.Error(err))
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}

	if *discoverManagementVirtualNodes {
		if err := doManagementVirtualNodeDiscovery(context.Background()); err != nil {
			logger.With(zap.Error(err)).Error("Failed to discover Management VirtualNodes")
		}
	}

	if *discoverManagementNodes {
		if err := doManagementNodeDiscovery(context.Background()); err != nil {
			logger.With(zap.Error(err)).Error("Failed to discover Management Nodes")
		}
	}

	if *populateManagementSwitchCredentials {
		if err := doManagementSwitchCredentials(context.Background()); err != nil {
			logger.With(zap.Error(err)).Error("Failed to populate Management Switch credentials")
		}
	}

	if *discoverRiver {
		doRiverDiscovery()
	}

	if *discoverMountain {
		doMountainDiscovery()
	}

	if *rediscoverFailedRedfishEndpoints {

		// At this point we should take advantage of the fact that we know all this information about the system and try
		// to fix any discovery attempts that have gone poorly.
		var potentiallyDiscoverableEndpoints []string

		notDiscoveredOKEndpoints := getNotDiscoveredOKEndpointFromHSM()
		logger.Debug("Endpoints with last discovery status not equal to DiscoverOK.",
			zap.Any("notDiscoveredOKEndpoints", notDiscoveredOKEndpoints))

		var notDiscoveredXnames []string
		var endpointWaitGroup sync.WaitGroup

		for _, endpoint := range notDiscoveredOKEndpoints {
			notDiscoveredXnames = append(notDiscoveredXnames, endpoint.ID)

			endpointWaitGroup.Add(1)

			go func(endpoint rf.RedfishEPDescription) {
				defer endpointWaitGroup.Done()

				// Check to see if it's Redfish is endpoint is reachable.
				reachableErr := checkBMCRedfish(endpoint.ID, endpoint.FQDN)
				if reachableErr != nil {
					logger.Warn("BMC is not reachable, ignoring for now.",
						zap.Error(reachableErr),
						zap.String("xname", endpoint.ID),
						zap.String("fqdn", endpoint.FQDN))
				} else {
					logger.Info("BMC is reachable and Redfish is responsive.",
						zap.String("xname", endpoint.ID),
						zap.String("fqdn", endpoint.FQDN))

					potentiallyDiscoverableEndpoints = append(potentiallyDiscoverableEndpoints, endpoint.ID)
				}
			}(endpoint)
		}

		endpointWaitGroup.Wait()

		if len(potentiallyDiscoverableEndpoints) > 0 {
			reDiscoverEndpoints(potentiallyDiscoverableEndpoints)
		} else {
			if len(notDiscoveredOKEndpoints) > 0 {
				logger.Info("HSM contains undiscovered Redfish endpoints, however none are reachable.",
					zap.Strings("notDiscoveredOKEndpoints", notDiscoveredXnames))
			} else {
				logger.Info("All Redfish endpoints in HSM are already discovered.")
			}
		}

	}
	logger.Info("HMS Discovery process complete.")
}
