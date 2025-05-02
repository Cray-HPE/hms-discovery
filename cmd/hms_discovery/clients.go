// MIT License
//
// (C) Copyright [2023,2025] Hewlett Packard Enterprise Development LP
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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	base "github.com/Cray-HPE/hms-base/v2"
	sls_common "github.com/Cray-HPE/hms-sls/v2/pkg/sls-common"
	rf "github.com/Cray-HPE/hms-smd/v2/pkg/redfish"
	"github.com/Cray-HPE/hms-xname/xnametypes"
	"github.com/hashicorp/go-retryablehttp"
)

var ErrNotFound = fmt.Errorf("not found")

func buildRequestURL(baseURL, path string, params map[string]string) string {
	url := fmt.Sprintf("%s/%s", baseURL, path)
	if len(params) > 0 {
		paramStrings := []string{}
		for key, value := range params {
			paramStrings = append(paramStrings, fmt.Sprintf("%s=%s", key, value))
		}

		url = fmt.Sprintf("%s?%s", url, strings.Join(paramStrings, "&"))
	}

	return url
}

//
// SLS
//

func getSLSSearchHardware(ctx context.Context, params map[string]string) (map[string]sls_common.GenericHardware, error) {
	url := buildRequestURL(*slsURL, "v1/search/hardware", params)

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to build GET request"), err)
	}
	// base.SetHTTPUserAgent(req, insta)

	response, err := httpClient.Do(req)
	defer base.DrainAndCloseResponseBody(response)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to perform GET request against SLS"), err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code %d expected 200", response.StatusCode)
	}

	var result []sls_common.GenericHardware
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Convert to a map for ease of use
	resultMap := map[string]sls_common.GenericHardware{}
	for _, hardware := range result {
		resultMap[hardware.Xname] = hardware
	}
	return resultMap, nil
}

//
// HSM
//

func getHSMStateComponents(ctx context.Context, params map[string]string) (map[string]base.Component, error) {
	url := buildRequestURL(*hsmURL, "State/Components", params)

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to build GET request"), err)
	}
	// base.SetHTTPUserAgent(req, insta)

	response, err := httpClient.Do(req)
	defer base.DrainAndCloseResponseBody(response)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to perform GET request against HSM"), err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code %d expected 200", response.StatusCode)
	}

	var result base.ComponentArray
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Convert to a map for ease of use
	resultMap := map[string]base.Component{}
	for _, component := range result.Components {
		resultMap[component.ID] = *component
	}

	return resultMap, nil
}

func postHSMStateComponent(ctx context.Context, component base.Component) error {
	if !xnametypes.IsHMSCompIDValid(component.ID) {
		return fmt.Errorf("invalid component ID (%s)", component.ID)
	}

	payload := base.ComponentArray{
		Components: []*base.Component{&component},
	}

	rawRequestBody, err := json.Marshal(payload)
	if err != nil {
		return errors.Join(fmt.Errorf("failed to marshal component object to json"), err)
	}

	url := fmt.Sprintf("%s/State/Components", *hsmURL)
	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", url, rawRequestBody)
	if err != nil {
		return errors.Join(fmt.Errorf("failed to build GET request"), err)
	}
	// base.SetHTTPUserAgent(request, sc.instanceName)

	response, err := httpClient.Do(req)
	defer base.DrainAndCloseResponseBody(response)
	if err != nil {
		return errors.Join(fmt.Errorf("failed to perform POST request against HSM"), err)
	}

	// If HSM sends back a response, then we should read the contents of the body so the Istio sidecar doesn't fill up
	var responseString string
	if response.Body != nil {
		responseRaw, _ := io.ReadAll(response.Body)
		responseString = string(responseRaw)
	}

	if response.StatusCode != 204 {
		return fmt.Errorf("unexpected status code %d expected 204: %s", response.StatusCode, responseString)
	}

	return nil
}

func getHSMInventoryRedfishEndpoint(ctx context.Context, xname string) (rf.RedfishEPDescription, error) {
	if !xnametypes.IsHMSCompIDValid(xname) {
		return rf.RedfishEPDescription{}, fmt.Errorf("invalid component ID (%s)", xname)
	}

	url := fmt.Sprintf("%s/Inventory/RedfishEndpoints/%s", *hsmURL, xname)
	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return rf.RedfishEPDescription{}, errors.Join(fmt.Errorf("failed to build GET request"), err)
	}

	// base.SetHTTPUserAgent(request, sc.instanceName)

	response, err := httpClient.Do(req)
	defer base.DrainAndCloseResponseBody(response)
	if err != nil {
		return rf.RedfishEPDescription{}, errors.Join(fmt.Errorf("failed to perform GET request against HSM"), err)
	}

	if response.StatusCode != 200 {
		// If HSM sends back a response, then we should read the contents of the body so the Istio sidecar doesn't fill up
		var responseString string
		if response.Body != nil {
			responseRaw, _ := io.ReadAll(response.Body)
			responseString = string(responseRaw)
		}

		if response.StatusCode == 404 {
			return rf.RedfishEPDescription{}, ErrNotFound
		} else if response.StatusCode != 200 {
			return rf.RedfishEPDescription{}, fmt.Errorf("unexpected status code %d expected 200: %s", response.StatusCode, responseString)
		}
	}

	var result rf.RedfishEPDescription
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return rf.RedfishEPDescription{}, err
	}

	return result, nil
}
