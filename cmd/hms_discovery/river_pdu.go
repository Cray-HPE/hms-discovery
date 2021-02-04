// MIT License
//
// (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
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
