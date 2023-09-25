// MIT License
//
// (C) Copyright [2023] Hewlett Packard Enterprise Development LP
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
