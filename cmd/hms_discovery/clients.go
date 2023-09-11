package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	base "github.com/Cray-HPE/hms-base"
	sls_common "github.com/Cray-HPE/hms-sls/v2/pkg/sls-common"
	"github.com/Cray-HPE/hms-xname/xnametypes"
	"github.com/hashicorp/go-retryablehttp"
)

//
// SLS
//

func getSLSManagementVirtualNodes(ctx context.Context) (map[string]sls_common.GenericHardware, error) {
	url := fmt.Sprintf("%s/v1/search/hardware?type=comptype_virtual_node&extra_properties.Role=Management", *slsURL)

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to build GET request"), err)
	}
	// base.SetHTTPUserAgent(req, insta)

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to perform GET request against SLS"), err)
	}
	defer response.Body.Close()

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

func getHSManagementVirtualNodes(ctx context.Context) (map[string]base.Component, error) {
	url := fmt.Sprintf("%s/State/Components?Type=VirtualNode&Role=Management", *hsmURL)
	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to build GET request"), err)
	}
	// base.SetHTTPUserAgent(req, insta)

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to perform GET request against HSM"), err)
	}
	defer response.Body.Close()

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
	if err != nil {
		return errors.Join(fmt.Errorf("failed to perform POST request against HSM"), err)
	}
	defer response.Body.Close()

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
