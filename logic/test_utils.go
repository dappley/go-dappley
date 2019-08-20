package logic

import (
	"strings"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic/laccount"
	"github.com/dappley/go-dappley/storage"
)

//get all addresses
func GetAllAddressesByPath(path string) ([]account.Address, error) {
	fl := storage.NewFileLoader(path)
	am := laccount.NewAccountManager(fl)
	err := am.LoadFromFile()
	if err != nil {
		return nil, err
	}

	addresses := am.GetAddresses()

	return addresses, err
}

func GetTestAccountPath() string {
	return strings.Replace(laccount.GetAccountFilePath(), "accounts", "accounts_test", -1)
}
