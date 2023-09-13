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

package switches

import (
	"errors"
	"fmt"

	securestorage "github.com/Cray-HPE/hms-securestorage"
)

/*
 * This file really shouldn't even be here, but REDS doesn't have any of the credential stuff in an exported place.
 */

type RedsCredStore struct {
	CCPath string
	SS     securestorage.SecureStorage
}

type SwitchCredentials struct {
	SNMPUsername     string
	SNMPAuthPassword string
	SNMPPrivPassword string
}

func (switchCredentials SwitchCredentials) String() string {
	return fmt.Sprintf("SNMPUsername: %s, SNMPAuthPassword: <REDACTED>, SNMPPrivPassword: <REDACTED>",
		switchCredentials.SNMPUsername)
}

type RedsCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (redsCred RedsCredentials) String() string {
	return fmt.Sprintf("Username: %s, Password: <REDACTED>", redsCred.Username)
}

// Create a new RedsCredStore struct that uses a SecureStorage backing store.
func NewRedsCredStore(keyPath string, ss securestorage.SecureStorage) *RedsCredStore {
	ccs := &RedsCredStore{
		CCPath: keyPath,
		SS:     ss,
	}
	return ccs
}

// GetDefaultCredentials retrieves a map of default credentials, keyed by vendor,
// from a secure credentials store.
func (ccs *RedsCredStore) GetDefaultCredentials() (map[string]RedsCredentials, error) {
	credMapRtn := make(map[string]RedsCredentials)
	err := ccs.SS.Lookup(ccs.CCPath+"/defaults", &credMapRtn)

	return credMapRtn, err
}

// StoreDefaultCredentials stores a map of default credentials, keyed by vendor.
func (ccs *RedsCredStore) StoreDefaultCredentials(credentials map[string]RedsCredentials) error {
	err := ccs.SS.Store(ccs.CCPath+"/defaults", credentials)

	if err != nil {
		return errors.New("unable to store default credentials: " + err.Error())
	}
	return nil
}

// StoreDefaultCredentials stores a map of default credentials, keyed by vendor.
func (ccs *RedsCredStore) GetDefaultSwitchCredentials() (credentials SwitchCredentials, err error) {
	err = ccs.SS.Lookup(ccs.CCPath+"/switch_defaults", &credentials)

	return
}

// StoreDefaultCredentials stores a map of default credentials, keyed by vendor.
func (ccs *RedsCredStore) StoreDefaultSwitchCredentials(credentials SwitchCredentials) error {
	err := ccs.SS.Store(ccs.CCPath+"/switch_defaults", credentials)

	if err != nil {
		return errors.New("unable to store default switch credentials: " + err.Error())
	}
	return nil
}
