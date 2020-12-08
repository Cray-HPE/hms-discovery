package switches

import (
	"fmt"
	securestorage "stash.us.cray.com/HMS/hms-securestorage"
)

/*
 * This file really shouldn't even be here, but REDS doesn't have any of the credential stuff in an exported place.
 */

type RedsCredStore struct {
	CCPath string
	SS     securestorage.SecureStorage
}

type SwitchCredentials struct {
	SNMPUsername string
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

func (ccs *RedsCredStore) GetDefaultCredentials() (map[string]RedsCredentials, error) {
	credMapRtn := make(map[string]RedsCredentials)
	err := ccs.SS.Lookup(ccs.CCPath+"/defaults", &credMapRtn)

	return credMapRtn, err
}

func (ccs *RedsCredStore) GetDefaultSwitchCredentials() (credentials SwitchCredentials, err error) {
	err = ccs.SS.Lookup(ccs.CCPath+"/switch_defaults", &credentials)

	return
}
