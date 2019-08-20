package logic

import (
	"strings"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
)

//get all addresses
func GetAllAddressesByPath(path string) ([]core.Address, error) {
	fl := storage.NewFileLoader(path)
	wm := client.NewWalletManager(fl)
	err := wm.LoadFromFile()
	if err != nil {
		return nil, err
	}

	addresses := wm.GetAddresses()

	return addresses, err
}

func GetTestWalletPath() string {
	return strings.Replace(client.GetWalletFilePath(), "wallets", "wallets_test", -1)
}
