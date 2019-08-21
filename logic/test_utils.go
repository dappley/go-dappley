package logic

import (
	"strings"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic/account_logic"
	"github.com/dappley/go-dappley/storage"
)

//get all addresses
func GetAllAddressesByPath(path string) ([]account.Address, error) {
	fl := storage.NewFileLoader(path)
	am := account_logic.NewAccountManager(fl)
	err := am.LoadFromFile()
	if err != nil {
		return nil, err
	}

	addresses := am.GetAddresses()

	return addresses, err
}

func GetTestAccountPath() string {
	return strings.Replace(account_logic.GetAccountFilePath(), "accounts", "accounts_test", -1)
}
