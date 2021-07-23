// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package wallet

import (
	"bytes"
	"os"
	"sync"

	"github.com/dappley/go-dappley/core/account"
	accountpb "github.com/dappley/go-dappley/core/account/pb"
	errorValues "github.com/dappley/go-dappley/errors"
	"github.com/dappley/go-dappley/storage"
	laccountpb "github.com/dappley/go-dappley/wallet/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const accountDataPath = "../bin/accounts.dat"

type AccountManager struct {
	Accounts   []*account.Account
	fileLoader storage.FileStorage
	PassPhrase []byte
	mutex      sync.Mutex
}

//GetAccountFilePath return account file Path
func GetAccountFilePath() string {
	createAccountFile(accountDataPath)
	return accountDataPath
}

func createAccountFile(path string) {
	binFolder := "../bin"
	if !Exists(binFolder) {
		err := os.Mkdir(binFolder, os.ModePerm)
		if err != nil {
			logger.Errorf("Create account file folder. binFolder: %v, error: %v", binFolder, err.Error())
		}
	}
	if !Exists(path) {
		file, err := os.Create(path)
		file.Close()
		if err != nil {
			logger.Errorf("Create account file error: %v", err.Error())
		}
	}
}

func NewAccountManager(fileLoader storage.FileStorage) *AccountManager {
	return &AccountManager{
		fileLoader: fileLoader,
	}
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func (am *AccountManager) LoadFromFile() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	fileContent, err := am.fileLoader.ReadFromFile()

	if err != nil {
		return err
	}
	amProto := &laccountpb.AccountManager{}
	err = proto.Unmarshal(fileContent, amProto)

	if err != nil {
		return err
	}

	am.FromProto(amProto)
	return nil
}

func (am *AccountManager) IsEmpty() bool {
	return len(am.Accounts) == 0
}

// SaveAccountToFile saves Accounts to a file
func (am *AccountManager) SaveAccountToFile() {
	var content bytes.Buffer
	am.mutex.Lock()
	defer am.mutex.Unlock()
	rawBytes, err := proto.Marshal(am.ToProto())
	if err != nil {
		logger.WithError(err).Error("AccountManager: Save account to file failed")
		return
	}
	content.Write(rawBytes)
	am.fileLoader.SaveToFile(content)
}

func (am *AccountManager) AddAccount(account *account.Account) {
	am.mutex.Lock()
	am.Accounts = append(am.Accounts, account)
	am.mutex.Unlock()
}

func (am *AccountManager) GetAddresses() []account.Address {
	var addresses []account.Address

	am.mutex.Lock()
	defer am.mutex.Unlock()
	for _, account := range am.Accounts {
		addresses = append(addresses, account.GetAddress())
	}

	return addresses
}

func (am *AccountManager) GetAddressesWithPassphrase(password string) ([]string, error) {
	var addresses []string

	am.mutex.Lock()
	err := bcrypt.CompareHashAndPassword(am.PassPhrase, []byte(password))
	if err != nil {
		am.mutex.Unlock()
		return nil, errorValues.ErrPasswordIncorrect
	}
	for _, account := range am.Accounts {
		address := account.GetAddress().String()
		addresses = append(addresses, address)
	}
	am.mutex.Unlock()
	return addresses, nil
}

func (am *AccountManager) GetKeyPairByAddress(address account.Address) *account.KeyPair {
	account := am.GetAccountByAddress(address)
	if account == nil {
		return nil
	}
	return account.GetKeyPair()
}

func (am *AccountManager) GetAccountByAddress(address account.Address) *account.Account {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	for _, account := range am.Accounts {
		if account.GetAddress() == address {
			return account
		}
	}
	return nil
}

func (am *AccountManager) GetAccountByAddressWithPassphrase(address account.Address, password string) (*account.Account, error) {
	err := bcrypt.CompareHashAndPassword(am.PassPhrase, []byte(password))
	if err == nil {
		account := am.GetAccountByAddress(address)
		if account == nil {
			return nil, errorValues.ErrAddressNotFound
		}
		return account, nil
	}
	return nil, errorValues.ErrPasswordIncorrect

}

func (am *AccountManager) ToProto() proto.Message {
	pbAccounts := []*accountpb.Account{}
	for _, account := range am.Accounts {
		pbAccounts = append(pbAccounts, account.ToProto().(*accountpb.Account))
	}

	return &laccountpb.AccountManager{
		Accounts:   pbAccounts,
		PassPhrase: am.PassPhrase,
	}
}

func (am *AccountManager) FromProto(pb proto.Message) {
	accounts := []*account.Account{}
	for _, accountPb := range pb.(*laccountpb.AccountManager).Accounts {
		account := &account.Account{}
		account.FromProto(accountPb)
		accounts = append(accounts, account)
	}

	am.Accounts = accounts
	am.PassPhrase = pb.(*laccountpb.AccountManager).PassPhrase

}
