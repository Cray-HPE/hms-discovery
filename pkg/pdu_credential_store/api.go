// Copyright 2019-2020 Hewlett Packard Enterprise Development LP

package pdu_credential_store

import (
	"errors"
	"path"
)

// CredentialsGlobalKey is the Vault key used to access RTS global credentials
const CredentialsGlobalKey = "global/pdu"

func (credStore *PDUCredentialStore) SetKeypathValue(data map[string]interface{}) (err error) {
	err = credStore.SecureStorage.Store(credStore.KeyPath, data)

	return
}

func (credStore *PDUCredentialStore) GetDefaultPDUCredentails() (cred DefaultCredential, err error) {
	key := path.Join(credStore.KeyPath, CredentialsGlobalKey)
	err = credStore.SecureStorage.Lookup(key, &cred)
	if err != nil {
		return
	}

	if cred.Password == "" || cred.Username == "" {
		err = errors.New("empty username or password")
	}
	return
}

func (credStore *PDUCredentialStore) StoreDefaultPDUCredentails(cred DefaultCredential) error {
	if cred.Password == "" || cred.Username == "" {
		return errors.New("empty username or password")
	}

	key := path.Join(credStore.KeyPath, CredentialsGlobalKey)
	return credStore.SecureStorage.Store(key, cred)
}

func (credStore *PDUCredentialStore) StorePDUCredentails(cred Device) error {
	if cred.Xname == "" {
		return errors.New("empty xname")
	}

	if cred.URL == "" {
		return errors.New("empty device url")
	}

	if cred.Password == "" || cred.Username == "" {
		return errors.New("empty username or password")
	}

	key := path.Join(credStore.KeyPath, cred.Xname)
	return credStore.SecureStorage.Store(key, cred)
}
