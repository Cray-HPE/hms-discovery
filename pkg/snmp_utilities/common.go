// MIT License
//
// (C) Copyright [2021] Hewlett Packard Enterprise Development LP
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

package snmp_utilities

import (
	"errors"
	"fmt"
	"github.com/k-sone/snmpgo"
	"github.com/Cray-HPE/hms-discovery/pkg/switches"
	"strconv"
	"strings"
)

// The OID which has the model number of the switch
var OIDModelNumber string = "1.3.6.1.2.1.47.1.1.1.1.13.2"

// The OID which maps ifIndexes to human-readable names
var OIDifIndexPortNameMap string = "1.3.6.1.2.1.31.1.1.1.1"

// The OID which maps physical port numbers to ifIndexes
var OIDPortNumberifIndex string = "1.3.6.1.2.1.17.1.4.1.2"

// The OID for the mac address table (with VLANs - should be there on all switches)
var OIDMacAddressesWithVLAN string = "1.3.6.1.2.1.17.7.1.2.2.1.2"

// The OID for the source information for VLANs
var OIDMacAddressSourceWithVLAN string = "1.3.6.1.2.1.17.7.1.2.2.1.3"

// The OID for NON-VLAN mac address table.  Only valid if the switch is
// configured with "enable-dot1d-mibwalk" first!
var OIDMACAddressesNoVLAN = "1.3.6.1.2.1.17.4.3.1.2"

// The OID for non-VLAN learned-mac sources.  Also only valid if configured
// with "enable-dot1d-mibwalk".
var OIDMacAddressSourceNoVLAN = "1.3.6.1.2.1.17.4.3.1.3"

// OID returned if an authentication fialure occurs.
var OIDAuthFailure string = "1.3.6.1.6.3.15.1.1.5.0"

// OID for querying the full name and version identification of
//a switches software operating-system and networking software.
var OIDSysDescr string = "1.3.6.1.2.1.1.1.0"

func GetSNMPOjbect(managementSwitch switches.ManagementSwitch) (snmp *snmpgo.SNMP, err error) {
	// Check that the address ends in a port number (required by goSNMP).
	if !strings.Contains(managementSwitch.Address, ":") {
		managementSwitch.Address = fmt.Sprintf("%s:161", managementSwitch.Address)
	}

	var securityLevel snmpgo.SecurityLevel
	var authType snmpgo.AuthProtocol
	var privType snmpgo.PrivProtocol

	if strings.ToLower(managementSwitch.SNMPAuthProtocol) == "none" {
		securityLevel = snmpgo.NoAuthNoPriv
	} else if strings.ToLower(managementSwitch.SNMPPrivProtocol) == "none" {
		securityLevel = snmpgo.AuthNoPriv
	} else {
		securityLevel = snmpgo.AuthPriv
	}

	if securityLevel != snmpgo.NoAuthNoPriv {
		if strings.ToLower(managementSwitch.SNMPAuthProtocol) == "md5" {
			authType = snmpgo.Md5
		} else if strings.ToLower(managementSwitch.SNMPAuthProtocol) == "sha" {
			authType = snmpgo.Sha
		}
	}

	if securityLevel == snmpgo.AuthPriv {
		if strings.ToLower(managementSwitch.SNMPPrivProtocol) == "aes" {
			privType = snmpgo.Aes
		} else if strings.ToLower(managementSwitch.SNMPPrivProtocol) == "des" {
			privType = snmpgo.Des
		}
	}

	snmp, err = snmpgo.NewSNMP(snmpgo.SNMPArguments{
		Version:       snmpgo.V3,
		Address:       managementSwitch.Address,
		Retries:       5,
		UserName:      managementSwitch.SNMPUser,
		SecurityLevel: securityLevel,
		AuthPassword:  managementSwitch.SNMPAuthPassword,
		AuthProtocol:  authType,
		PrivPassword:  managementSwitch.SNMPPrivPassword,
		PrivProtocol:  privType,
	})

	return
}

func snmpGetBulk(snmp *snmpgo.SNMP, oid string) (snmpgo.Pdu, error) {
	oids, err := snmpgo.NewOids([]string{oid})
	if err != nil {
		return nil, err
	}

	err = snmp.Open()
	if err != nil {
		return nil, err
	}
	defer snmp.Close()

	result, err := snmp.GetBulkWalk(oids, 0, 10)
	if err != nil {
		return nil, err
	}

	if result.ErrorStatus() != snmpgo.NoError {
		return nil, errors.New(result.ErrorStatus().String())
	}

	return result, nil
}

func getDynamicMacs(snmp *snmpgo.SNMP, useVLANs bool) (macPortMap map[string]int, err error) {
	var portSrc string
	if useVLANs {
		portSrc = OIDMacAddressesWithVLAN
	} else {
		portSrc = OIDMACAddressesNoVLAN
	}
	port, bulkErr := snmpGetBulk(snmp, portSrc)
	if bulkErr != nil {
		err = fmt.Errorf("unable to get port src: %w", bulkErr)
		return
	}

	// Process the MAC->port list into a map.
	macPortMap = make(map[string]int)
	for _, portEntry := range port.VarBinds() {
		portMac, conversionErr := MacAddressFromOID(portEntry.Oid.String())
		if conversionErr != nil {
			err = fmt.Errorf("failed to parse OID (%s) into MAC address: %w",
				portEntry.Oid.String(), conversionErr)
			continue
		}

		portNum, conversionErr := portEntry.Variable.BigInt()
		if conversionErr != nil {
			err = fmt.Errorf("failed to turn port number (%s) into an integer: %w",
				portEntry.Variable.String(), conversionErr)
		}

		if int((*portNum).Int64()) != 0 {
			macPortMap[portMac] = int((*portNum).Int64())
		}
	}

	return
}

func MacAddressFromOID(OID string) (macAddress string, err error) {
	OIDParts := strings.Split(OID, ".")
	if len(OIDParts) < 6 {
		err = errors.New("oid has fewer than 6 parts so it cannot contain a MAC address")
		return
	}

	for _, part := range OIDParts[len(OIDParts)-6:] {
		val, conversionErr := strconv.Atoi(part)
		if conversionErr != nil {
			err = fmt.Errorf("failed to convert part to int: %w", conversionErr)
			return
		}

		if val > 255 || val < 0 {
			err = fmt.Errorf("%s is >255 or <0 which is invalid", part)
			return
		}

		str := strconv.FormatInt(int64(val), 16)
		if len(str) < 2 {
			str = "0" + str
		}

		macAddress += str
	}

	return
}
