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

	base "github.com/Cray-HPE/hms-base/v2"
	compcredentials "github.com/Cray-HPE/hms-compcredentials"
	sls_common "github.com/Cray-HPE/hms-sls/v2/pkg/sls-common"
	"github.com/Cray-HPE/hms-xname/xnametypes"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

func doManagementNodeDiscovery(ctx context.Context) error {
	//
	// Gather information from SLS and HSM
	//

	// Query SLS for Management Nodes
	logger.Info("Querying SLS for Management Nodes")
	slsNodes, err := getSLSSearchHardware(ctx, map[string]string{
		"type":                  sls_common.Node.String(),
		"extra_properties.Role": base.RoleManagement.String(),
	})
	if err != nil {
		return errors.Join(fmt.Errorf("failed to retrieve Management Nodes from SLS"), err)
	}

	// Query HSM for Management Nodes
	logger.Info("Querying HSM for Management Nodes")
	hsmNodes, err := getHSMStateComponents(ctx, map[string]string{
		"Type": xnametypes.Node.String(),
		"Role": base.RoleManagement.String(),
	})
	if err != nil {
		return errors.Join(fmt.Errorf("failed to retrieve Management Nodes from HSM"), err)
	}

	//
	// Determine differences between SLS and HSM Components
	//
	missingFromHSM := map[string]bool{}
	missingFromSLS := map[string]bool{}
	presentInBoth := map[string]bool{}

	for xname := range slsNodes {
		if _, exists := hsmNodes[xname]; exists {
			// Exists in both
			presentInBoth[xname] = true
		} else {
			// Present in SLS but not HSM
			missingFromHSM[xname] = true
		}
	}

	for xname := range hsmNodes {
		if _, exists := slsNodes[xname]; !exists {
			// Present in HSM, but not SLS
			missingFromSLS[xname] = true
		}
	}

	logger.With(zap.Strings("xnames", xnameSetToSlice(presentInBoth))).Info("Management Nodes present in both SLS and HSM State Components")
	logger.With(zap.Strings("xnames", xnameSetToSlice(missingFromHSM))).Info("Management Nodes present in SLS, missing from HSM State Components")
	logger.With(zap.Strings("xnames", xnameSetToSlice(missingFromSLS))).Info("Management Nodes present in HSM State Components, missing from SLS")

	defaultCreds, err := redsCredentialStore.GetDefaultCredentials()
	if err != nil {
		return errors.Join(fmt.Errorf("unable to get default credentials"), err)
	}

	//
	// Populate HSM with missing managment nodes
	//
	for nodeXname := range missingFromHSM {
		bmcXname := xnametypes.GetHMSCompParent(nodeXname)
		subLogger := logger.With(zap.String("nodeXname", nodeXname), zap.String("bmcXname", bmcXname))

		slsNode := slsNodes[nodeXname]
		var slsExtraProperties sls_common.ComptypeNode
		if err := mapstructure.Decode(slsNode.ExtraPropertiesRaw, &slsExtraProperties); err != nil {
			subLogger.With(zap.Any("slsNode", slsNode), zap.Error(err)).Error("Failed to decode node extra properties")
			continue
		}

		// Check to see if redfish endpoint exists
		if _, err := getHSMInventoryRedfishEndpoint(ctx, bmcXname); err == nil {
			// Redfish endpoint exists in HSM skip it, no work to do.
			subLogger.Info("Management Node BMC exists in Redfish Endpoints, but node not in State Components. Waiting for rediscovery")
			continue
		} else if !errors.Is(err, ErrNotFound) {
			subLogger.With(zap.Error(err)).Error("Failed to query HSM for Redfish Endpoint")
			continue
		}

		// It does not exist, need to create information in HSM

		mgmtSwitchConnectors, err := getSLSSearchHardware(ctx, map[string]string{
			"node_nics": bmcXname,
		})
		if err != nil {
			subLogger.With(zap.Error(err)).Error("Failed to query SLS for BMC's MgmtSwitchConnector")
			continue
		}

		subLogger.With(
			zap.Strings("mgmtSwitchConnectors", xnameMapToSlice(mgmtSwitchConnectors)),
		).Debug("Found Management Switch Connections")

		subLogger.Debug("Creating redfish endpoint for Management Node BMC")
		// First check to see if there are credentials in Vault for this xname. If there are we won't
		// re-set them in case they've been changed from the defaults.
		credentials, err := hsmCredentialStore.GetCompCred(bmcXname)
		if err != nil {
			subLogger.With(zap.Error(err)).
				Error("Unable to check Vault for BMC credentials, not creating RedfishEndpoint in HSM")
			continue
		}

		if credentials.Username == "" || credentials.Password == "" {
			credentials := compcredentials.CompCredentials{
				Xname:    bmcXname,
				Username: defaultCreds["Cray"].Username,
				Password: defaultCreds["Cray"].Password,
			}

			err = hsmCredentialStore.StoreCompCred(credentials)
			if err != nil {
				subLogger.With(zap.Error(err)).
					Error("Unable to set credentials, not creating RedfishEndpoint in HSM")
				continue
			}
			subLogger.Debug("Set BMC credentials in Vault")
		} else {
			subLogger.Debug("BMC credentials already exist in Vault")
		}

		if err := informHSM(bmcXname, bmcXname, ""); err != nil {
			subLogger.With(zap.Error(err)).Error("Failed to inform HSM about Management Node BMC")
		}

		subLogger.Info("Created RedfishEndpoint in HSM for Management Node BMC")

		if len(mgmtSwitchConnectors) == 0 && slsExtraProperties.SubRole == "Master" {
			subLogger.Debug("Management Node BMC has no connection to HMN, creating component in HSM")

			component := base.Component{
				ID:      nodeXname,
				State:   base.StatePopulated.String(),
				Role:    slsExtraProperties.Role,
				SubRole: slsExtraProperties.SubRole,
				NID:     json.Number(fmt.Sprintf("%d", slsExtraProperties.NID)),
				NetType: base.NetSling.String(),
				Arch:    base.ArchX86.String(),
			}

			subLogger.With(zap.Any("component", component)).Debug("Component to be created")

			if err := postHSMStateComponent(ctx, component); err != nil {
				subLogger.With(zap.Any("slsVirtualNode", slsNode), zap.Error(err)).Error("Failed to create State component for Management VirtualNode")
				continue
			}

			subLogger.Info("Created Component in HSM for Management Node")
		}
	}

	return nil
}
