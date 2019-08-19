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

package account_logic

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/core/account"
	accountpb "github.com/dappley/go-dappley/core/account/pb"
	laccountpb "github.com/dappley/go-dappley/logic/account_logic/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const accountConfigFilePath = "../core/account/account.conf"

var (
	ErrPasswordIncorrect = errors.New("password is incorrect")
	ErrAddressNotFound   = errors.New("address not found in local accounts")
)

type AccountManager struct {
	Accounts   []*account.Account
	fileLoader storage.FileStorage
	PassPhrase []byte
	mutex      sync.Mutex
	timer      time.Timer
	Locked     bool
}

//GetAccountFilePath return account file Path
func GetAccountFilePath() string {
	conf := &accountpb.AccountConfig{}
	if Exists(accountConfigFilePath) {
		config.LoadConfig(accountConfigFilePath, conf)
	} else if Exists(strings.Replace(accountConfigFilePath, "..", "../..", 1)) {
		config.LoadConfig(strings.Replace(accountConfigFilePath, "..", "../..", 1), conf)
	}

	if conf == nil {
		return ""
	}
	accountPath := strings.Replace(conf.GetFilePath(), "/accounts.dat", "", 1)
	var accountfile string
	if Exists(accountPath) {
		accountfile, _ = filepath.Abs(conf.GetFilePath())
	} else if Exists(strings.Replace(accountPath, "..", "../..", 1)) {
		accountfile, _ = filepath.Abs(strings.Replace(conf.GetFilePath(), "..", "../..", 1))
	}
	return accountfile
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

func (am *AccountManager) NewTimer(timeout time.Duration) {
	am.timer = *time.NewTimer(timeout)
}

func (am *AccountManager) LoadFromFile() error {

	am.mutex.Lock()
	fileContent, err := am.fileLoader.ReadFromFile()

	if err != nil {
		am.mutex.Unlock()
		am.SaveAccountToFile()
		fileContent, err = am.fileLoader.ReadFromFile()
		am.mutex.Lock()
	}
	amProto := &laccountpb.AccountManager{}
	err = proto.Unmarshal(fileContent, amProto)

	if err != nil {
		return err
	}

	am.FromProto(amProto)
	am.mutex.Unlock()
	return nil
}

func (am *AccountManager) IsFileEmpty() (bool, error) {
	fileContent, err := am.fileLoader.ReadFromFile()
	if err != nil {
		return true, err
	}
	return len(fileContent) == 0, nil

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

func RemoveAccountFile() {
	conf := &accountpb.AccountConfig{}
	config.LoadConfig(accountConfigFilePath, conf)
	if conf == nil {
		return
	}
	os.Remove(strings.Replace(conf.GetFilePath(), "accounts", "accounts_test", -1))
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
		addresses = append(addresses, account.GetKeyPair().GenerateAddress())
		subkeys := account.GetSubKeys()
		for _, subkey := range subkeys {
			addresses = append(addresses, subkey.GenerateAddress())
		}
	}

	return addresses
}

func (am *AccountManager) GetAddressesWithPassphrase(password string) ([]string, error) {
	var addresses []string

	am.mutex.Lock()
	err := bcrypt.CompareHashAndPassword(am.PassPhrase, []byte(password))
	if err != nil {
		am.mutex.Unlock()
		return nil, ErrPasswordIncorrect
	}
	for _, account := range am.Accounts {
		address := account.GetKeyPair().GenerateAddress().String()
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
		for _, key := range account.GetAllKeys() {
			if key.GenerateAddress() == address {
				return account
			}
		}
	}
	return nil
}

func (am *AccountManager) GetAccountByAddressWithPassphrase(address account.Address, password string) (*account.Account, error) {
	err := bcrypt.CompareHashAndPassword(am.PassPhrase, []byte(password))
	if err == nil {
		account := am.GetAccountByAddress(address)
		if account == nil {
			return nil, ErrAddressNotFound
		}
		return account, nil

	}
	return nil, ErrPasswordIncorrect

}

func (am *AccountManager) SetUnlockTimer(timeout time.Duration) {
	am.Locked = false
	am.SaveAccountToFile()
	am.NewTimer(timeout)
	am.timer.Reset(timeout)
	go am.UnlockExpire()
}

func (am *AccountManager) UnlockExpire() {
	defer am.timer.Stop()
	select {
	case <-am.timer.C:
		am.LoadFromFile()
		am.Locked = true
		am.SaveAccountToFile()
	}
}

func (am *AccountManager) ToProto() proto.Message {
	pbAccounts := []*accountpb.Account{}
	for _, account := range am.Accounts {
		pbAccounts = append(pbAccounts, account.ToProto().(*accountpb.Account))
	}

	return &laccountpb.AccountManager{
		Accounts:   pbAccounts,
		PassPhrase: am.PassPhrase,
		Locked:     am.Locked,
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
	am.Locked = pb.(*laccountpb.AccountManager).Locked
}
