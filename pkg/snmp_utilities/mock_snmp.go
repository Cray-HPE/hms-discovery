package snmp_utilities

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// This just exists to make development easier. Testing with a real switch is a pain, so it's nice to just capture
// the output from a switch one, dump it into some files, and then just pass those files right back when testing.

func (snmpInterface MockSNMP) GetPortMap() (portMap map[int]string, err error) {
	jsonFile, err := os.Open("configs/portMap.json")
	if err != nil {
		return
	}

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	var mockPortMap map[string]map[int]string
	err = json.Unmarshal(jsonBytes, &mockPortMap)

	if err != nil {
		return
	}

	var found bool
	portMap, found = mockPortMap[snmpInterface.SwitchXname]

	if !found {
		err = fmt.Errorf("switch xname not found")
	}

	return
}

func (snmpInterface MockSNMP) GetPortNumberMap() (portNumberMap map[int]int, err error) {
	jsonFile, err := os.Open("configs/portNumberMap.json")
	if err != nil {
		return
	}

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	var mockPortNumberMap map[string]map[int]int
	err = json.Unmarshal(jsonBytes, &mockPortNumberMap)

	if err != nil {
		return
	}

	var found bool
	portNumberMap, found = mockPortNumberMap[snmpInterface.SwitchXname]

	if !found {
		err = fmt.Errorf("switch xname not found")
	}

	return
}

func (snmpInterface MockSNMP) GetMACPortNameTable(map[int]int, map[int]string) (macPortMap map[string]string,
	err error) {
	jsonFile, err := os.Open("configs/macPortMap.json")
	if err != nil {
		return
	}

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	var mockPortMap map[string]map[string]string
	err = json.Unmarshal(jsonBytes, &mockPortMap)

	if err != nil {
		return
	}

	var found bool
	macPortMap, found = mockPortMap[snmpInterface.SwitchXname]

	if !found {
		err = fmt.Errorf("switch xname not found")
	}

	return
}
