package logic

import (
	"strings"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/storage"
)

//get all addresses
func GetAllAddressesByPath(path string) ([]client.Address, error) {
	fl := storage.NewFileLoader(path)
	am := client.NewAccountManager(fl)
	err := am.LoadFromFile()
	if err != nil {
		return nil, err
	}

	addresses := am.GetAddresses()

	return addresses, err
}

func GetTestAccountPath() string {
	return strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1)
}
