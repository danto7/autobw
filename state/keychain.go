package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/danto7/autobw/build"
	"github.com/keybase/go-keychain"
)

const service = "autobw"
const accessGroup = "autobw.danto7.com.github"

var account = "default"
var label = "autobw session"

func init() {
	if build.Debug {
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

type State struct {
	BitwardenSession string        `json:"bitwarden_session"`
	LastUnlock       time.Time     `json:"last_unlock"`
	UnlockTimeout    time.Duration `json:"unlock_timeout"`
}

var ErrorNotFound = errors.New("Could not find existing state")

func (s *State) Load() error {
	data, err := keychain.GetGenericPassword(service, account, label, accessGroup)
	if err == keychain.ErrorItemNotFound || len(data) == 0 {
		return ErrorNotFound
	} else if err != nil {
		return fmt.Errorf("error getting password from keychain: %w", err)
	}

	err = json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("error unmarshalling keychain data: %w", err)
	}

	return nil
}

func (s *State) Write() error {
	item := buildItem()

	data, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	item.SetData(data)
	if build.Debug {
		// set to always accessible for debugging
		// otherwise the keychain needs to be unlocked after every recompile on macos
		item.SetAccessible(keychain.AccessibleAlways)
	}
	err = keychain.AddItem(item)
	if !errors.Is(err, keychain.ErrorDuplicateItem) {
		return err
	}

	slog.Debug("item already exists, update it")
	query := buildItem()
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(false)
	return keychain.UpdateItem(query, item)
}
