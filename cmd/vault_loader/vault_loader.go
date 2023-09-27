// MIT License
//
// (C) Copyright [2019, 2021, 2023] Hewlett Packard Enterprise Development LP
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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Cray-HPE/hms-discovery/pkg/switches"
	securestorage "github.com/Cray-HPE/hms-securestorage"
)

func main() {
	var secureStorage securestorage.SecureStorage
	var credStorage *switches.RedsCredStore

	defaultNodeCredentials, ok := os.LookupEnv("VAULT_REDFISH_BMC_DEFAULTS")
	if !ok {
		panic("Value not set for VAULT_REDFISH_BMC_DEFAULTS")
	}

	defaultSwitchCredentials, ok := os.LookupEnv("VAULT_REDFISH_SWITCH_DEFAULTS")
	if !ok {
		panic("Value not set for VAULT_REDFISH_SWITCH_DEFAULTS")
	}

	// Setup Vault. It's kind of a big deal, so we'll wait forever for this to work.
	fmt.Println("Connecting to Vault...")
	for {
		var err error
		// Start a connection to Vault
		if secureStorage, err = securestorage.NewVaultAdapter(""); err != nil {
			fmt.Printf("Unable to connect to Vault (%s)...trying again in 5 seconds.\n", err)
			time.Sleep(5 * time.Second)
		} else {
			fmt.Println("Connected to Vault")
			credStorage = switches.NewRedsCredStore("secret/reds-creds", secureStorage)
			break
		}
	}

	// Node BMC defaults
	var nodeCredentials map[string]switches.RedsCredentials
	err := json.Unmarshal([]byte(defaultNodeCredentials), &nodeCredentials)
	if err != nil {
		fmt.Printf("Unable to unmarshal defaults for node BMCs: %s", err)
	}

	// Uncomment the following lines if you want to debug what is getting put into Vault.
	//prettyCredentials, _ := json.MarshalIndent(nodeCredentials, "\t", "   ")
	//fmt.Printf("Loading:\n\t%s\n\n", prettyCredentials)

	err = credStorage.StoreDefaultCredentials(nodeCredentials)
	if err != nil {
		fmt.Printf("Unable to store defaults for node BMCs: %s", err)
	}

	// Switch defaults
	var switchCredentials switches.SwitchCredentials
	err = json.Unmarshal([]byte(defaultSwitchCredentials), &switchCredentials)
	if err != nil {
		fmt.Printf("Unable to unmarshal defaults for switches: %s", err)
	}

	// Uncomment the following lines if you want to debug what is getting put into Vault.
	//prettyCredentials, _ = json.MarshalIndent(switchCredentials, "\t", "   ")
	//fmt.Printf("Loading:\n\t%s\n\n", prettyCredentials)

	err = credStorage.StoreDefaultSwitchCredentials(switchCredentials)
	if err != nil {
		fmt.Printf("Unable to store defaults for switches: %s", err)
	}

	fmt.Println("Done.")
}
