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
	"sort"

	base "github.com/Cray-HPE/hms-base/v2"
	sls_common "github.com/Cray-HPE/hms-sls/v2/pkg/sls-common"
	"github.com/Cray-HPE/hms-xname/xnametypes"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

func xnameSetToSlice(in map[string]bool) []string {
	xnames := []string{}
	for xname, exists := range in {
		if exists {
			xnames = append(xnames, xname)
		}
	}

	sort.Strings(xnames)
	return xnames
}

func xnameMapToSlice(in map[string]sls_common.GenericHardware) []string {
	xnames := []string{}
	for xname := range in {
		xnames = append(xnames, xname)
	}

	sort.Strings(xnames)
	return xnames
}

func doManagementVirtualNodeDiscovery(ctx context.Context) error {
	//
	// Gather information from SLS and HSM
	//

	// Query SLS for Management Virtual Nodes
	logger.Info("Querying SLS for Management VirtualNodes")
	slsVirtualNodes, err := getSLSSearchHardware(ctx, map[string]string{
		"type":                  sls_common.VirtualNode.String(),
		"extra_properties.Role": base.RoleManagement.String(),
	})
	if err != nil {
		return errors.Join(fmt.Errorf("failed to retrieve Management VirtualNodes from SLS"), err)
	}

	// Query HSM for Management Virtual Node
	logger.Info("Querying HSM for Management VirtualNodes")
	hsmVirtualNodes, err := getHSMStateComponents(ctx, map[string]string{
		"Type": xnametypes.VirtualNode.String(),
		"Role": base.RoleManagement.String(),
	})
	if err != nil {
		return errors.Join(fmt.Errorf("failed to retrieve Management VirtualNodes from HSM"), err)
	}

	//
	// Determine differences between SLS and HSM
	//
	missingFromHSM := map[string]bool{}
	missingFromSLS := map[string]bool{}
	presentInBoth := map[string]bool{}

	for xname := range slsVirtualNodes {
		if _, exists := hsmVirtualNodes[xname]; exists {
			// Exists in both
			presentInBoth[xname] = true
		} else {
			// Present in SLS but not HSM
			missingFromHSM[xname] = true
		}
	}

	for xname := range hsmVirtualNodes {
		if _, exists := slsVirtualNodes[xname]; !exists {
			// Present in HSM, but not SLS
			missingFromSLS[xname] = true
		}
	}

	logger.With(zap.Strings("xnames", xnameSetToSlice(presentInBoth))).Info("Management VirtualNodes present in both SLS and HSM")
	logger.With(zap.Strings("xnames", xnameSetToSlice(missingFromHSM))).Info("Management VirtualNodes present in SLS, missing from HSM")
	logger.With(zap.Strings("xnames", xnameSetToSlice(missingFromSLS))).Info("Management VirtualNodes present in HSM, missing from SLS")

	// Create State components for hardware
	for xname := range missingFromHSM {
		subLogger := logger.With(zap.String("xname", xname))
		slsVirtualNode := slsVirtualNodes[xname]
		subLogger.With(zap.Any("slsVirtualNode", slsVirtualNode)).Debug("Processing SLS Virtual Node")

		var slsExtraProperties sls_common.ComptypeVirtualNode
		if err := mapstructure.Decode(slsVirtualNode.ExtraPropertiesRaw, &slsExtraProperties); err != nil {
			subLogger.With(zap.Any("slsVirtualNode", slsVirtualNode), zap.Error(err)).Error("Failed to decode virtual node extra properties")
			continue
		}

		component := base.Component{
			ID:      xname,
			State:   base.StateStandby.String(), // Use StandBy state to allow HBTD to transistion component states correctly. If using populate HBTD would not be able to tranistion the state, as a force is needed to transition to a different state.
			Role:    slsExtraProperties.Role,
			SubRole: slsExtraProperties.SubRole,
			NID:     json.Number(fmt.Sprintf("%d", slsExtraProperties.NID)),
			NetType: base.NetSling.String(),
			Arch:    base.ArchX86.String(),
		}

		if err := postHSMStateComponent(ctx, component); err != nil {
			subLogger.With(zap.Any("slsVirtualNode", slsVirtualNode), zap.Error(err)).Error("Failed to create State component for Management VirtualNode")
			continue
		}

		subLogger.Info("Created Virtual Node in HSM State Components")
	}

	return nil
}
