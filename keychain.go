package main

import (
	"errors"
	"log/slog"

	"github.com/keybase/go-keychain"
)

const service = "autobw"
const accessGroup = "autobw.danto7.com.github"

var account = "default"
var label = "autobw session"

func init() {
	if DEBUG {
		label += " (debug)"
		account += " (debug)"
	}
}
func buildItem() keychain.Item {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(account)
	item.SetLabel(label)
	item.SetAccessGroup(accessGroup)
	return item
}

func updateSecret(data []byte) error {
	item := buildItem()
	item.SetData(data)
	if DEBUG {
		// set to always accessible for debugging
		// otherwise the keychain needs to be unlocked after every recompile on macos
		item.SetAccessible(keychain.AccessibleAlways)
	}
	err := keychain.AddItem(item)
	slog.Debug("Error during add item", "err", err)
	if !errors.Is(err, keychain.ErrorDuplicateItem) {
		return err
	}

	// item already exists, update it
	query := buildItem()
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(false)
	return keychain.UpdateItem(query, item)
}

func getSecret() ([]byte, error) {
	return keychain.GetGenericPassword(service, account, label, accessGroup)
}
