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
	"fmt"
	"strconv"
	"strings"
)

func (snmpInterface RealSNMP) GetPortMap() (portMap map[int]string, err error) {
	result, bulkErr := snmpGetBulk(snmpInterface.SNMP, OIDifIndexPortNameMap)
	if bulkErr != nil {
		err = fmt.Errorf("failed to perform bulk get: %w", bulkErr)
		return
	}

	portMap = make(map[int]string)

	for _, res := range result.VarBinds() {
		oidParts := strings.Split(res.Oid.String(), ".")
		strIndex := oidParts[len(oidParts)-1]
		ifIndex, convertErr := strconv.Atoi(strIndex)
		if convertErr != nil {
			err = fmt.Errorf("failed to convert ifIndex to integer: %w", convertErr)
			return
		}

		portMap[ifIndex] = res.Variable.String()
	}

	return
}

func (snmpInterface RealSNMP) GetPortNumberMap() (portNumberMap map[int]int, err error) {
	result, bulkErr := snmpGetBulk(snmpInterface.SNMP, OIDPortNumberifIndex)
	if bulkErr != nil {
		err = fmt.Errorf("failed to perform bulk get: %w", bulkErr)
		return
	}

	portNumberMap = make(map[int]int)

	for _, res := range result.VarBinds() {
		oidParts := strings.Split(res.Oid.String(), ".")
		strPortID := oidParts[len(oidParts)-1]
		portID, convertErr := strconv.Atoi(strPortID)
		if convertErr != nil {
			err = fmt.Errorf("failed to convert port ID to integer: %w", convertErr)
			return
		}

		keyBI, err := res.Variable.BigInt()
		if err != nil {
			return nil, err
		}

		key := int(keyBI.Int64())

		portNumberMap[key] = portID
	}

	return
}

func (snmpInterface RealSNMP) GetMACPortNameTable(portNumberIfIndexMap map[int]int,
	ifIndexPortNameMap map[int]string) (macPortMap map[string]string, err error) {
	portMap, portMapErr := getDynamicMacs(snmpInterface.SNMP, false)
	if portMapErr != nil {
		err = fmt.Errorf("failed to get non-VLAN MAC port map: %w", portMapErr)
		return
	}

	tempPortMap, portMapErr := getDynamicMacs(snmpInterface.SNMP, true)
	if portMapErr != nil {
		err = fmt.Errorf("failed to get VLAN MAC port map: %w", portMapErr)
		return
	}

	// Combine both the VLAN and non-VLAN port maps into a single map.
	for key, val := range tempPortMap {
		if _, ok := portMap[key]; !ok {
			portMap[key] = val
		}
	}

	macPortMap = make(map[string]string)
	for key, value := range portMap {
		ifIndex, ok := portNumberIfIndexMap[value]
		if !ok {
			err = fmt.Errorf("failed to map port (%d) to ifIndex", value)
			return
		}

		name, ok := ifIndexPortNameMap[ifIndex]
		if !ok {
			err = fmt.Errorf("failed to map ifIndex (%d) to port name", ifIndex)
			return
		}

		macPortMap[key] = name
	}

	return
}

