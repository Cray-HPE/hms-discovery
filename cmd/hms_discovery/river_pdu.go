// Copyright 2020 Hewlett Packard Enterprise Development LP

package main

import (
	"fmt"

	"go.uber.org/zap"
	"stash.us.cray.com/HMS/hms-discovery/pkg/pdu_credential_store"
	"stash.us.cray.com/HMS/hms-smd/pkg/sm"
)

func informRTS(xname, fqdn, macWithoutPunctuation string, unknownComponent sm.CompEthInterface) error {
	// Get Default Credentails for the PDU
	defaultCreds, err := pduCredentialStore.GetDefaultPDUCredentails()
	if err != nil {
		return fmt.Errorf("failed to get default PDU credentials: %w", err)
	}

	// From here on we know the xname
	unknownComponent.CompID = xname

	// Add the new ethernet interface.
	addErr := dhcpdnsClient.AddNewEthernetInterface(unknownComponent, true)
	if addErr != nil {
		logger.Error("Failed to add new ethernet interface to HSM, not processing further!",
			zap.Error(addErr), zap.Any("unknownComponent", unknownComponent))
		return addErr
	}

	logger.Info("Updated ethernet interface in HSM.",
		zap.Any("unknownComponent", unknownComponent))

	// PDU Device Credentails
	device := pdu_credential_store.Device{
		Xname:    xname,
		URL:      fmt.Sprintf("https://%s/jaws", fqdn),
		Username: defaultCreds.Username,
		Password: defaultCreds.Password,
	}

	// ...and finally tell RTS about the newly found PDU.
	err = pduCredentialStore.StorePDUCredentails(device)
	if err != nil {
		return fmt.Errorf("failed to store PDU credentails: %w", err)
	}

	logger.Info("Informed RTS of discovered PDU via Vault",
		zap.String("xname", xname),
		zap.String("fqdn", fqdn),
		zap.String("device.url", device.URL),
	)

	return nil
}
