package main

import (
	"context"
	"errors"
	"fmt"

	base "github.com/Cray-HPE/hms-base"
	compcredentials "github.com/Cray-HPE/hms-compcredentials"
	sls_common "github.com/Cray-HPE/hms-sls/v2/pkg/sls-common"
	"github.com/Cray-HPE/hms-xname/xnametypes"
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
	// Determine differences between SLS and HSM
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

	logger.With(zap.Strings("xnames", xnameSetToSlice(presentInBoth))).Info("Management Nodes present in both SLS and HSM")
	logger.With(zap.Strings("xnames", xnameSetToSlice(missingFromHSM))).Info("Management Nodes present in SLS, missing from HSM")
	logger.With(zap.Strings("xnames", xnameSetToSlice(missingFromSLS))).Info("Management Nodes present in HSM, missing from SLS")

	//
	// Determine which
	//
	defaultCreds, err := redsCredentialStore.GetDefaultCredentials()
	if err != nil {
		return errors.Join(fmt.Errorf("Unable to get default credentials"), err)
	}

	for nodeXname := range missingFromHSM {
		bmcXname := xnametypes.GetHMSCompParent(nodeXname)
		subLogger := logger.With(zap.String("nodeXname", nodeXname), zap.String("bmcXname", bmcXname))

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

		if len(mgmtSwitchConnectors) > 0 {
			subLogger.Info("Management Node BMC has connection to HMN")
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

		} else {
			subLogger.Info("Management Node BMC has no connection to HMN")
		}
	}

	return nil
}
