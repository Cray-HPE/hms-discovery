// MIT License
//
// (C) Copyright [2019, 2021] Hewlett Packard Enterprise Development LP
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
