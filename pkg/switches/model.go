package switches

import "fmt"

type ManagementSwitch struct {
	Xname            string
	Aliases          []string
	Address          string
	SNMPUser         string
	SNMPAuthPassword string
	SNMPAuthProtocol string
	SNMPPrivPassword string
	SNMPPrivProtocol string
	Model            string
}

func (s ManagementSwitch) String() string {
	return fmt.Sprintf("{Xname: %s, Aliases: %s, Model: %s, Address: %s, SNMP User: %s, "+
		"SNMP Auth Password: <REDACTED>, SNMP Auth Protocol: %s, "+
		"SNMP Priv Password: <REDACTED>, SNMP Priv Protocol: %s}",
		s.Xname, s.Aliases, s.Model, s.Address, s.SNMPUser, s.SNMPAuthProtocol, s.SNMPPrivProtocol)
}
