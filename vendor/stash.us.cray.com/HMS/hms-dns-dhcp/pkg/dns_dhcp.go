// Copyright 2019 Cray Inc.

package dns_dhcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"stash.us.cray.com/HMS/hms-smd/pkg/sm"
)

type DNSDHCPHelper struct {
    HSMURL string
    HTTPClient *retryablehttp.Client
}

func NewDHCPDNSHelper(HSMURL string, HTTPClient *retryablehttp.Client) (helper DNSDHCPHelper) {
    helper.HSMURL = HSMURL

    if HTTPClient != nil {
        helper.HTTPClient = HTTPClient
    } else {
        helper.HTTPClient = retryablehttp.NewClient()
    }

    return
}

func (helper *DNSDHCPHelper) GetUnknownComponents() (unknownComponents []sm.CompEthInterface, err error) {
    url := fmt.Sprintf("%s/Inventory/EthernetInterfaces?ComponentID", helper.HSMURL)

    response, err := helper.HTTPClient.Get(url)
    if err != nil {
        return
    }

    if response.StatusCode != http.StatusOK {
        err = fmt.Errorf("unexpected status code from HSM: %d", response.StatusCode)
        return
    }

    jsonBytes, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return
    }
    defer response.Body.Close()

    err = json.Unmarshal(jsonBytes, &unknownComponents)

    return
}

func (helper *DNSDHCPHelper) GetAllEthernetInterfaces() (unknownComponents []sm.CompEthInterface, err error) {
    url := fmt.Sprintf("%s/Inventory/EthernetInterfaces", helper.HSMURL)

    response, err := helper.HTTPClient.Get(url)
    if err != nil {
        return
    }

    if response.StatusCode != http.StatusOK {
        err = fmt.Errorf("unexpected status code from HSM: %d", response.StatusCode)
        return
    }

    jsonBytes, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return
    }
    defer response.Body.Close()

    err = json.Unmarshal(jsonBytes, &unknownComponents)

    return
}

func (helper *DNSDHCPHelper) AddNewEthernetInterface(newInterface sm.CompEthInterface, patchIfConflict bool) (
    err error) {
    payloadBytes, marshalErr := json.Marshal(newInterface)
    if marshalErr != nil {
        err = fmt.Errorf("failed to marshal interface: %w", marshalErr)
        return
    }

    url := fmt.Sprintf("%s/Inventory/EthernetInterfaces", helper.HSMURL)

    request, requestErr := retryablehttp.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
    if requestErr != nil {
        err = fmt.Errorf("failed to construct request: %w", requestErr)
        return
    }
    request.Header.Set("Content-Type", "application/json")

    response, doErr := helper.HTTPClient.Do(request)
    if doErr != nil {
        err = fmt.Errorf("failed to execute POST request: %w", doErr)
        return
    }

    if response.StatusCode == http.StatusConflict {
        if patchIfConflict {
            err = helper.PatchEthernetInterface(newInterface)
        } else {
            err = fmt.Errorf("failed to add new interface because it already exists")
        }
    } else if response.StatusCode != http.StatusCreated {
        err = fmt.Errorf("unexpected status code (%d): %s", response.StatusCode, response.Status)
    }

    return
}

func (helper *DNSDHCPHelper) PatchEthernetInterface(theInterface sm.CompEthInterface) (err error) {
    payloadBytes, marshalErr := json.Marshal(theInterface)
    if marshalErr != nil {
        err = fmt.Errorf("failed to marshal interface: %w", marshalErr)
        return
    }

    macID := strings.ReplaceAll(theInterface.MACAddr, ":", "")
    url := fmt.Sprintf("%s/Inventory/EthernetInterfaces/%s", helper.HSMURL, macID)

    request, requestErr := retryablehttp.NewRequest("PATCH", url, bytes.NewBuffer(payloadBytes))
    if requestErr != nil {
        err = fmt.Errorf("failed to construct request: %w", requestErr)
        return
    }
    request.Header.Set("Content-Type", "application/json")

    response, doErr := helper.HTTPClient.Do(request)
    if doErr != nil {
        err = fmt.Errorf("failed to execute PATCH request: %w", doErr)
        return
    }

    if response.StatusCode != http.StatusOK {
        err = fmt.Errorf("unexpected status code (%d): %s", response.StatusCode, response.Status)
    }

    return
}
