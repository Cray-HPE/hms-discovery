package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sls_common "github.com/Cray-HPE/hms-sls/v2/pkg/sls-common"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

func doManagementSwitchCredentials(ctx context.Context) error {
	logger.Info("Populating Management Switch credentials in vault")

	//
	// Find management switches in SLS
	//
	allManagementSwitches := map[string]sls_common.GenericHardware{}
	for _, switchType := range []sls_common.HMSStringType{sls_common.MgmtSwitch, sls_common.MgmtHLSwitch, sls_common.CDUMgmtSwitch} {
		logger.Sugar().Debugf("Querying SLS for %s Management Switches", switchType)

		foundSwitches, err := getSLSSearchHardware(ctx, map[string]string{
			"type": string(switchType),
		})
		if err != nil {
			return errors.Join(fmt.Errorf("failed to search SLS for %s hardware", switchType), err)
		}

		for xname, foundSwitch := range foundSwitches {
			allManagementSwitches[xname] = foundSwitch
		}
	}

	logger.With(zap.Strings("xnames", xnameMapToSlice(allManagementSwitches))).Debug("Found Management Switches")

	//
	// Populate Vault with credentials
	//

	defaultCreds, err := redsCredentialStore.GetDefaultSwitchCredentials()
	if err != nil {
		return errors.Join(fmt.Errorf("failed to retrieve default switch SNMP credentials from vault"), err)
	}

	for xname, slsSwitch := range allManagementSwitches {
		subLogger := logger.With(zap.String("xname", xname))

		//
		// Parse switch extra properties
		//

		var slsExtraProperties sls_common.ComptypeMgmtSwitch
		if err := mapstructure.Decode(slsSwitch.ExtraPropertiesRaw, &slsExtraProperties); err != nil {
			subLogger.With(zap.Any("slsSwitch", slsSwitch), zap.Error(err)).Error("Failed to decode switch extra properties")
			continue
		}

		//
		// Populate Vault with credentials
		//

		// Check to see if credentials exist
		switchCred, err := hsmCredentialStore.GetCompCred(xname)
		if err != nil {
			subLogger.With(zap.Error(err)).Error("failed to query vault for switch credentials")
			continue
		}

		if switchCred.SNMPAuthPass != "" && switchCred.SNMPPrivPass != "" && switchCred.Username != "" {
			subLogger.Debug("Found switch creds in vault")
			continue
		}

		// If we get nothing back from Vault then we need to push something in.
		switchCred.Xname = xname

		vaultURIPrefix := "vault://"
		if slsExtraProperties.SNMPAuthPassword != "" && !strings.HasPrefix(slsExtraProperties.SNMPAuthPassword, vaultURIPrefix) {
			switchCred.SNMPAuthPass = slsExtraProperties.SNMPAuthPassword
		} else {
			switchCred.SNMPAuthPass = defaultCreds.SNMPAuthPassword
		}

		if slsExtraProperties.SNMPPrivPassword != "" && !strings.HasPrefix(slsExtraProperties.SNMPPrivPassword, vaultURIPrefix) {
			switchCred.SNMPPrivPass = slsExtraProperties.SNMPPrivPassword
		} else {
			switchCred.SNMPPrivPass = defaultCreds.SNMPPrivPassword
		}

		if slsExtraProperties.SNMPUsername != "" {
			switchCred.Username = slsExtraProperties.SNMPUsername
		} else {
			switchCred.Username = defaultCreds.SNMPUsername
		}

		err = hsmCredentialStore.StoreCompCred(switchCred)
		if err != nil {
			subLogger.With(zap.Error(err)).Error("Unable to store credentials for switch")
			continue
		}

		subLogger.Info("Stored credential for switch")
	}

	return nil
}
