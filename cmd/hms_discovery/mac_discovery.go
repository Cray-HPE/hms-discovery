// MIT License
//
// (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
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

/*
This implements mac-based discovery for both River and Mputnain nodes. It
examiines smd's EthernetInterfaes table for unknown MAC addresses. When it
finds one, it first checks if it's a mountain endpoint (distinguished by the
mac address prefix). If not, it checks the switches to see if it's a river
mac.

Once it finds the mac, it identifies it. For Mountain nodes, this is a reverse
engineering of the MAC address. For river, it examines the switches and
determines where it is plugged in, then compares that with the map of cables
in SLS.
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
	base "stash.us.cray.com/HMS/hms-base"
	compcredentials "stash.us.cray.com/HMS/hms-compcredentials"
	"stash.us.cray.com/HMS/hms-discovery/pkg/snmp_utilities"
	"stash.us.cray.com/HMS/hms-discovery/pkg/switches"
	sls_common "stash.us.cray.com/HMS/hms-sls/pkg/sls-common"
	rf "stash.us.cray.com/HMS/hms-smd/pkg/redfish"
	"stash.us.cray.com/HMS/hms-smd/pkg/sm"
)

const VaultPrefix = "vault://"

var defaultCredentials map[string]switches.RedsCredentials

func onComponentDiscovered(component sm.CompEthInterface, macWithoutPunctuation string) error {
	// Put the creds in Vault.
	compCred := compcredentials.CompCredentials{
		Xname:    component.CompID,
		Username: defaultCredentials["Cray"].Username,
		Password: defaultCredentials["Cray"].Password,
	}
	compCredErr := hsmCredentialStore.StoreCompCred(compCred)
	if compCredErr != nil {
		logger.Fatal("Failed to store BMC credentials!",
			zap.Error(compCredErr),
			zap.String("xname", component.CompID),
		)
	}

	// Add the new ethernet interface.
	addErr := dhcpdnsClient.AddNewEthernetInterface(component, true)

	if addErr != nil {
		logger.Error("Failed to add new ethernet interface to HSM, not processing further!",
			zap.Error(addErr), zap.Any("unknownComponent", component))
		return addErr
	} else {
		logger.Info("Updated ethernet interface in HSM.",
			zap.Any("unknownComponent", component))
	}

	// ...and finally tell HSM to go discover.
	informErr := informHSM(component.CompID, component.CompID, macWithoutPunctuation)
	if informErr != nil {
		logger.Error("Failed to notify HSM about endpoint!",
			zap.Error(informErr),
			zap.String("xname", component.CompID),
		)
	}

	return nil
}

/*
Attempts to convert a mac address from a mountain cabinet into an xname

Returns three things:
- xname (*string) - The xname associated with the mac. Nil on error or
	if no conversion could be performed
- err (error) - any error that occured during processing.

Note that a return vaule of (nil, nil) is possible and indicates the
mac address was a valid moutnain mac, but that it could not be converted.
this is most likely because it is not a nC, sC, or cC.
*/
func mountainMacToXname(mac string) (*string, error) {
	// We need to slice and dice this mac...
	/*
		PP:XX:XX:CC:SS:II

		PP: Prefix (fixed)
		XX:XX: Rack number
		CC: Chassis number
		SS: Slot
		II: Index

		slot number == 0: Chassis controller, idx must be 0
		slot number <48: ?? node, probably. We'll error if we hit this
		slot number >=48: Node card (subtract 48 to get real slot number)
		Slot number >= 96: switch card (subtract 96 from slot to get real slot
			number)

		Index should be right shifted by 4 to counteract a left shift by 4 when
		the mac is made.
	*/

	var err error

	// Some safety/sanity checks
	if len(mac) != 12 {
		return nil, fmt.Errorf("Passed mac %s is not a mac address: wrong length, should be 12, was",
			mac, len(mac))
	}

	if !strings.HasPrefix(mac, *mountainMacPrefix) {
		return nil, fmt.Errorf("Passed mac %s is not a Mountain mac; should have prefix %s",
			*mountainMacPrefix)
	}

	// We never use the prefix, so don't bother converting it.
	//prefix := strconv.ParseInt(mac[0:1], 16, 0)
	rack, err := strconv.ParseInt(mac[2:5], 16, 0)
	if err != nil {
		return nil, err
	}
	chassis, err := strconv.ParseInt(mac[6:7], 16, 0)
	if err != nil {
		return nil, err
	}
	slot, err := strconv.ParseInt(mac[8:9], 16, 0)
	if err != nil {
		return nil, err
	}
	idx, err := strconv.ParseInt(mac[10:11], 16, 0)
	if err != nil {
		return nil, err
	}

	// Figure out what kind of hardware this is so we can
	var res string
	if slot == 0 && idx == 0 {
		// Chassic controller (cC)
		res = fmt.Sprintf("x%dc%db0", rack, chassis)
	} else if slot >= 48 && slot < 96 {
		// Node Controller (nC)
		res = fmt.Sprintf("x%dc%ds%db%d", rack, chassis, slot-48, idx>>4)
	} else if slot >= 96 {
		// Switch Controller (sC)
		res = fmt.Sprintf("x%dc%dr%db0", rack, chassis, slot-96)
	} else {
		return nil, nil
	}
	return &res, nil
}

func doRiverDiscovery() {
	// Get the unknown components from HSM.
	unknownComponents, getErr := dhcpdnsClient.GetUnknownComponents()
	if getErr != nil {
		logger.Error("Unable to get unknown components!", zap.Error(getErr))
	}

	// Nothing to do here?
	if len(unknownComponents) == 0 {
		logger.Info("No unknown components to discover.")
		return
	}

	logger.Debug("Found undiscovered components, attempting to identify.",
		zap.Any("unknownComponents", unknownComponents))

	// Gather the default credentials for the BMCs in case we need them.
	var credsErr error
	defaultCredentials, credsErr = redsCredentialStore.GetDefaultCredentials()
	if credsErr != nil {
		// Unfortunately this has to be fatal as we can do all the rest of the work but if we can't authentication
		// it's for nothing.
		logger.Fatal("Failed to get default BMC credentials!", zap.Error(credsErr))
	}

	// Make sure we actually got something.
	if defaultCredentials["Cray"].Username == "" || defaultCredentials["Cray"].Password == "" {
		logger.Fatal("Default Cray credentials blank for either username or password!")
	}

	// Ah crap, somebody expects us to work I guess. Ok, let's get the info we need from the switches.
	managementSwitches, switchErr := getSwitches()
	if switchErr != nil {
		logger.Error("Unable to get switches!", zap.Error(switchErr))
	}

	// What we need is a mapping of all the switches by their name and their port mappings,
	// then we can process the unknown hardware.
	switchPortMapping := make(map[string]map[string]string)

	// For each of the switches get their port mappings. If any part fails for a given switch we won't call the
	// whole thing a failure and instead try to move on to the next switch.
	for _, managementSwitch := range managementSwitches {
		switchLogger := logger.With(
			zap.String("managementSwitchXname", managementSwitch.Xname),
			zap.Strings("managementSwitchAliases", managementSwitch.Aliases))

		snmp, snmpErr := snmp_utilities.GetSNMPOjbect(managementSwitch)
		if snmpErr != nil {
			switchLogger.Warn("Unable to get SNMP object for management switch!",
				zap.Error(snmpErr),
			)

			continue
		}

		switchLogger.Debug("Generated SNMP object for switch.", zap.Any("snmp", snmp))

		// Setup an instance of an inteface to use to get the rest of the data.
		var snmpInterface snmp_utilities.SNMPInterface
		snmpMode := os.Getenv("SNMP_MODE")
		if snmpMode == "MOCK" {
			snmpInterface = snmp_utilities.MockSNMP{
				SwitchXname: managementSwitch.Xname,
			}
			switchLogger.Debug("Using mock SNMP interface.")
		} else {
			snmpInterface = snmp_utilities.RealSNMP{
				SNMP: snmp,
			}
			switchLogger.Debug("Using production SNMP interface.")
		}

		// Get a mapping of interface indexes to names.
		portMap, portMapErr := snmpInterface.GetPortMap()
		if portMapErr != nil {
			switchLogger.Warn("Failed to get port map for management switch!",
				zap.Error(portMapErr),
			)

			continue
		}

		switchLogger.Debug("Got port map from switch.", zap.Any("portMap", portMap))

		// Next get a mapping of interface indexes to numbers.
		portNumberMap, portNumberMapErr := snmpInterface.GetPortNumberMap()
		if portNumberMapErr != nil {
			switchLogger.Fatal("Failed to get port number map for management switch!",
				zap.Error(portNumberMapErr),
			)

			continue
		}

		switchLogger.Debug("Got port number map from switch.", zap.Any("portNumberMap", portNumberMap))

		// Reverse the keys and values
		portNumberIfIndexMap := make(map[int]int)
		for key, val := range portNumberMap {
			portNumberIfIndexMap[val] = key
		}

		// Now get the MAC addresses for all the ports on this switch.
		macPortMap, macPortErr := snmpInterface.GetMACPortNameTable(portNumberIfIndexMap, portMap)
		if macPortErr != nil {
			switchLogger.Warn("Unable to get MAC to port mapping for switch!", zap.Error(macPortErr))

			continue
		}

		switchLogger.Debug("Got MAC port map from switch.", zap.Any("macPortMap", macPortMap))

		// Add this switch to the master map.
		switchPortMapping[managementSwitch.Xname] = macPortMap
	}

	// Keep track of the xnames we successfully and unsuccessfully process.
	var discoveredXnames []string
	var failedXnames []string
	var remainingUnknownComponents []sm.CompEthInterface

	// Finally we can process all of the unknown hardware.
	for _, unknownComponent := range unknownComponents {
		logger.Debug("Searching for unknown component.", zap.Any("unknownComponent", unknownComponent))

		macWithoutPunctuation := strings.ReplaceAll(unknownComponent.MACAddr, ":", "")

		globallyFound := false

		// This is a good place ot have a switch statement for river versus moutnain
		if strings.HasPrefix(macWithoutPunctuation, *mountainMacPrefix) {
			// This is a mountain MAC. Figure out what it is by name.
			xname, err := mountainMacToXname(macWithoutPunctuation)

			if err != nil {
				logger.Info("Identified MAC as mountain, but couldn't parse an xname.",
					zap.String("mac", macWithoutPunctuation),
					zap.Error(err),
				)
				break
			}

			unknownComponent.CompID = *xname

			err = onComponentDiscovered(unknownComponent, macWithoutPunctuation)
			if err != nil {
				failedXnames = append(failedXnames, unknownComponent.CompID)

				break
			}

			logger.Info("Successfully identified and informed HSM about Mountain endpoint.",
				zap.String("xname", unknownComponent.CompID),
			)

			globallyFound = true
			discoveredXnames = append(discoveredXnames, unknownComponent.CompID)

			break
		} else {
			// Find the switch and port this MAC belongs to.
			var port string
			var switchFound bool

			for managementSwitchXname, switchMacPortMap := range switchPortMapping {
				port, switchFound = switchMacPortMap[macWithoutPunctuation]

				// If this switch doesn't have this MAC continue to the next switch.
				if !switchFound {
					continue
				}

				logger.Info("Found MAC address in switch.",
					zap.String("macWithoutPunctuation", macWithoutPunctuation),
					zap.String("managementSwitchXname", managementSwitchXname),
					zap.String("port", port),
				)

				// Great, we found it! Now do a reverse lookup with SLS to figure out the identity.
				xname, slsErr := getXnameForSwitchPort(managementSwitchXname, port)
				if slsErr != nil {
					logger.Warn("Failed to lookup xname for switch/port combination.",
						zap.String("managementSwitchXname", managementSwitchXname),
						zap.String("port", port),
					)

					// If we fail that's not necessarily the end of the world. Since these are layer 2 networks it's
					// possible we'll see the same MAC in many different switches so we need to keep looking. In the end
					// only one switch will have the correct port mapping that corresponds to what SLS has.
					continue
				}

				// If we've made it here we know exactly what this BMC is. Therefore any failure from this point on will
				// be treated as "fatal" for this device rather than just a continue.

				if base.GetHMSType(xname) == base.CabinetPDUController {
					// ServerTech PDUs are discovered differently then other types of hardware, as they do no talk native Redfish.
					// TODO: In the future when we have PDU's that support redfish, we need to inform HSM instead of RTS

					if informErr := informRTS(xname, xname, macWithoutPunctuation, unknownComponent); informErr != nil {
						logger.Error("Failed to notify RTS about PDU!",
							zap.Error(informErr),
							zap.String("xname", xname),
						)

						failedXnames = append(failedXnames, xname)
						break
					}

					logger.Info("Successfully identified and informed RTS about PDU.",
						zap.String("xname", xname),
						zap.String("managementSwitchXname", managementSwitchXname),
						zap.String("port", port),
					)

					globallyFound = true
					discoveredXnames = append(discoveredXnames, xname)

					break
				}

				unknownComponent.CompID = xname

				err := onComponentDiscovered(unknownComponent, macWithoutPunctuation)
				if err != nil {
					failedXnames = append(failedXnames, xname)

					break
				}

				logger.Info("Successfully identified and informed HSM about endpoint.",
					zap.String("xname", xname),
					zap.String("managementSwitchXname", managementSwitchXname),
					zap.String("port", port),
				)

				globallyFound = true
				discoveredXnames = append(discoveredXnames, xname)

				break
			}
		}

		// Did we find this MAC in any switches successfully?
		if !globallyFound {
			logger.Error("MAC address in HSM not found in any switch!",
				zap.Any("unknownComponent", unknownComponent),
			)

			remainingUnknownComponents = append(remainingUnknownComponents, unknownComponent)
		}

	}

	logger.Info("River discovery finished.",
		zap.Strings("discoveredXnames", discoveredXnames),
		zap.Strings("failedXnames", failedXnames),
		zap.Any("remainingUnknownComponents", remainingUnknownComponents))
}

func checkBMCRedfish(xname string, fqdn string) (err error) {
	// Endpoint might require authentication, get what we need.
	creds, credsErr := hsmCredentialStore.GetCompCred(xname)
	if credsErr != nil {
		// Credential error...log it but don't stop, might not need the authentication.
		logger.Error("Failed to get credentials from Vault for xname!",
			zap.String("xname", xname), zap.Error(credsErr))
	}

	redfishURL := fmt.Sprintf("https://%s/redfish/v1/", fqdn)

	//req.SetBasicAuth(request.Auth.Username, request.Auth.Password)
	request, requestErr := retryablehttp.NewRequest("GET", redfishURL, nil)
	if requestErr != nil {
		err = fmt.Errorf("failed to make request: %w", requestErr)
		return
	}
	request.SetBasicAuth(creds.Username, creds.Password)

	response, doErr := httpClient.Do(request)
	if doErr != nil {
		err = fmt.Errorf("failed to execute GET request: %w", doErr)
		return
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code from Redfish: %d", response.StatusCode)
		return
	}

	return
}

func informHSM(xname string, fqdn string, macAddr string) (err error) {
	hsmEPDescritption := rf.RedfishEPDescription{
		ID:             xname,
		FQDN:           fqdn,
		MACAddr:        macAddr,
		RediscOnUpdate: true,
		Enabled:        true,
	}

	payloadBytes, marshalErr := json.Marshal(hsmEPDescritption)
	if marshalErr != nil {
		err = fmt.Errorf("failed to marshal HSM endpoint description: %w", marshalErr)
		return
	}

	hsmURL := fmt.Sprintf("%s/Inventory/RedfishEndpoints", *hsmURL)
	request, requestErr := retryablehttp.NewRequest("POST", hsmURL, bytes.NewBuffer(payloadBytes))
	if requestErr != nil {
		err = fmt.Errorf("failed to construct request: %w", requestErr)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	response, doErr := httpClient.Do(request)
	if doErr != nil {
		err = fmt.Errorf("failed to execute POST request: %w", doErr)
		return
	}

	if response.StatusCode == http.StatusConflict {
		// If we're in conflict (which is almost provably impossible given we start with unknown devices),
		// then PATCH the entry.
		request.Method = "PATCH"

		_, doErr := httpClient.Do(request)
		if doErr != nil {
			err = fmt.Errorf("failed to execute PATCH request: %w", doErr)
			return
		}
	} else if response.StatusCode != http.StatusCreated {
		err = fmt.Errorf("unexpected status code (%d): %s. Payload: %s",
			response.StatusCode, response.Status, string(payloadBytes))
	}

	return
}

func getXnameForSwitchPort(managementSwitchXname string, portName string) (xname string, err error) {
	url := fmt.Sprintf("%s/v1/search/hardware?type=comptype_mgmt_switch_connector&class=River"+
		"&extra_properties.VendorName=%s&parent=%s", *slsURL, portName, managementSwitchXname)

	response, err := httpClient.Get(url)
	if err != nil {
		return
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code from SLS: %d", response.StatusCode)
		return
	}

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	defer response.Body.Close()

	var genericHardware []sls_common.GenericHardware
	err = json.Unmarshal(jsonBytes, &genericHardware)

	// Make sure we have a result.
	if len(genericHardware) == 0 {
		err = fmt.Errorf("no results found for switch/port combination")
		return
	}

	// This should be impossible given the primary key would clash, but always good to check the corners.
	if len(genericHardware) > 1 {
		err = fmt.Errorf("found more than one match for a switch/port combination")
		return
	}

	// Now we're sure there's only 1 result and it's the one we're after.
	switchConnector := genericHardware[0]

	var switchConnectorProperties sls_common.ComptypeMgmtSwitchConnector
	decodeErr := mapstructure.Decode(switchConnector.ExtraPropertiesRaw, &switchConnectorProperties)
	if decodeErr != nil {
		err = fmt.Errorf("unable to decode switch connector properties: %w", decodeErr)
		return
	}

	// Make sure there is at least 1 NodeNic.
	if len(switchConnectorProperties.NodeNics) == 0 {
		err = fmt.Errorf("no NodeNics defined for switch connector")
		return
	}

	// That said, there should only be 1, any more than that and we have ambiguity.
	if len(switchConnectorProperties.NodeNics) > 1 {
		err = fmt.Errorf("more than one NodeNic for switch connector, can not determine xname")
		return
	}

	// Finally, the only thing left must be the name we were after.
	xname = switchConnectorProperties.NodeNics[0]

	return
}

func getSwitches() (managementSwitches []switches.ManagementSwitch, err error) {
	url := fmt.Sprintf("%s/v1/search/hardware?type=comptype_mgmt_switch&class=River", *slsURL)

	response, err := httpClient.Get(url)
	if err != nil {
		return
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code from SLS: %d", response.StatusCode)
		return
	}

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	defer response.Body.Close()

	var genericHardware []sls_common.GenericHardware
	unmarshalErr := json.Unmarshal(jsonBytes, &genericHardware)
	if unmarshalErr != nil {
		err = fmt.Errorf("unable to unmarshal generic hardware from SLS: %w", unmarshalErr)
		return
	}

	// Start by getting the default switch credentials.
	defaultSwitchCredentials, switchErr := redsCredentialStore.GetDefaultSwitchCredentials()
	if switchErr != nil {
		logger.Error("Unable to get default switch credentials!", zap.Error(switchErr))
	}

	// Build the list of switches from the generic hardware.
	for _, genericSwitch := range genericHardware {
		var switchProperties sls_common.ComptypeMgmtSwitch
		decodeErr := mapstructure.Decode(genericSwitch.ExtraPropertiesRaw, &switchProperties)
		if decodeErr != nil {
			// Might be a one off...don't quit over it.
			logger.Error("Unable to decode switch properties!", zap.Error(decodeErr))
			continue
		}

		// At this point we need to retrieve from Vault what we need.
		// Start by trying the hardware specific credentials.
		switchCreds, credErr := hsmCredentialStore.GetCompCred(genericSwitch.Xname)
		if credErr != nil {
			logger.Error("Unable to get credentials for switch!", zap.Error(credErr))
		}

		// The credentials we have for the specific hardware might be blank,
		// supplement with the defaults where necessary.
		if switchCreds.Username == "" {
			if switchProperties.SNMPUsername != "" &&
				!strings.HasPrefix(switchProperties.SNMPUsername, VaultPrefix) {
				switchCreds.Username = switchProperties.SNMPUsername
			} else {
				switchCreds.Username = defaultSwitchCredentials.SNMPUsername
			}
		}
		if switchCreds.SNMPPrivPass == "" {
			if switchProperties.SNMPPrivPassword != "" &&
				!strings.HasPrefix(switchProperties.SNMPPrivPassword, VaultPrefix) {
				switchCreds.SNMPPrivPass = switchProperties.SNMPPrivPassword
			} else {
				switchCreds.SNMPPrivPass = defaultSwitchCredentials.SNMPPrivPassword
			}
		}
		if switchCreds.SNMPAuthPass == "" {
			if switchProperties.SNMPAuthPassword != "" &&
				!strings.HasPrefix(switchProperties.SNMPAuthPassword, VaultPrefix) {
				switchCreds.SNMPAuthPass = switchProperties.SNMPAuthPassword
			} else {
				switchCreds.SNMPAuthPass = defaultSwitchCredentials.SNMPAuthPassword
			}
		}

		newSwitch := switches.ManagementSwitch{
			Xname:            genericSwitch.Xname,
			Aliases:          switchProperties.Aliases,
			Address:          switchProperties.IP4Addr,
			SNMPUser:         switchProperties.SNMPUsername,
			SNMPAuthPassword: switchCreds.SNMPAuthPass,
			SNMPAuthProtocol: switchProperties.SNMPAuthProtocol,
			SNMPPrivPassword: switchCreds.SNMPPrivPass,
			SNMPPrivProtocol: switchProperties.SNMPPrivProtocol,
			Model:            switchProperties.Model,
		}

		managementSwitches = append(managementSwitches, newSwitch)
	}

	return
}
