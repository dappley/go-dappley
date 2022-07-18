package logic

import (
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/wallet"
	logger "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
)

var AccountTestFileMutex = &sync.Mutex{}

//get all addresses
func GetAllAddressesByPath(path string) ([]account.Address, error) {
	am, err := GetAccountManager(path)
	if err != nil {
		return nil, err
	}

	addresses := am.GetAddresses()

	return addresses, err
}

func GetTestAccountPath() string {
	binFolder, _ := filepath.Split(wallet.GetAccountFilePath())
	testAccountPath := binFolder + "accounts_test.dat"
	if wallet.Exists(testAccountPath) {
		return testAccountPath
	} else {
		if !wallet.Exists(binFolder) {
			err := os.Mkdir(binFolder, os.ModePerm)
			if err != nil {
				logger.Errorf("Create test account file folder error: %v", err.Error())
			}
		}
		file, err := os.Create(testAccountPath)
		file.Close()
		if err != nil {
			logger.Errorf("Create test account file error: %v", err.Error())
		} else {
			return testAccountPath
		}
	}
	return ""
}

func RemoveAccountTestFile() {
	binFolder, _ := filepath.Split(wallet.GetAccountFilePath())
	testAccountPath := binFolder + "accounts_test.dat"
	os.Remove(testAccountPath)
}
