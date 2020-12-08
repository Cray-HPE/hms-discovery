package snmp_utilities

import "github.com/k-sone/snmpgo"

// Generic mock interface for testing.
type MockSNMP struct {
	SwitchXname string
}

// Real interface.
type RealSNMP struct {
	SNMP *snmpgo.SNMP
}

type SNMPInterface interface {
	GetPortMap() (portMap map[int]string, err error)
	GetPortNumberMap() (portNumberMap map[int]int, err error)
	GetMACPortNameTable(portNumberIfIndexMap map[int]int, ifIndexPortNameMap map[int]string) (
		macPortMap map[string]string, err error)
}
