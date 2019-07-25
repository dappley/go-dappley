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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dappley/go-dappley/client"
	logicpb "github.com/dappley/go-dappley/logic/pb"
	"github.com/dappley/go-dappley/storage"
	storage_mock "github.com/dappley/go-dappley/storage/mock"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAccountManager_LoadFromFileExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().ReadFromFile()

	am := NewAccountManager(mockStorage)
	am.LoadFromFile()

}

func TestAccountManager_LoadFromFileNotExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)

	gomock.InOrder(
		mockStorage.EXPECT().ReadFromFile().Return(nil, errors.New("err")),
		mockStorage.EXPECT().SaveToFile(gomock.Any()),
		mockStorage.EXPECT().ReadFromFile(),
	)

	am := NewAccountManager(mockStorage)
	am.LoadFromFile()
}

func TestAccountManager_SaveAccountToFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().SaveToFile(gomock.Any())
	am := NewAccountManager(mockStorage)
	am.SaveAccountToFile()

}

func TestAccountManager_SaveAccountToFile_with_passphrase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().SaveToFile(gomock.Any())
	am := NewAccountManager(mockStorage)
	account := client.NewAccount()
	am.Accounts = append(am.Accounts, account)
	am.SaveAccountToFile()

}

func TestAccountManager_AddAccount(t *testing.T) {
	am := NewAccountManager(nil)
	account := client.NewAccount()
	am.AddAccount(account)

	assert.Equal(t, account, am.Accounts[0])
}

func TestAccount_GetAddresses(t *testing.T) {
	am := NewAccountManager(nil)
	account := client.NewAccount()
	addresses := []client.Address{account.GetKeyPair().GenerateAddress()}
	am.Accounts = append(am.Accounts, account)
	assert.Equal(t, addresses, am.GetAddresses())
}

func TestAccount_GetAddressesNoAccount(t *testing.T) {
	am := NewAccountManager(nil)
	assert.Equal(t, []client.Address(nil), am.GetAddresses())
}

func TestAccountManager_GetAccountByAddress(t *testing.T) {
	am := NewAccountManager(nil)
	account := client.NewAccount()
	am.Accounts = append(am.Accounts, account)
	assert.Equal(t, account, am.GetAccountByAddress(account.GetKeyPair().GenerateAddress()))
}

func TestAccountManager_GetAccountByAddress_withPassphrase(t *testing.T) {
	am := NewAccountManager(nil)
	passPhrase, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.Equal(t, nil, err)
	am.PassPhrase = passPhrase
	am.Locked = true
	account := client.NewAccount()
	am.Accounts = append(am.Accounts, account)
	account1, err := am.GetAccountByAddressWithPassphrase(account.GetKeyPair().GenerateAddress(), "password")
	account2, err1 := am.GetAccountByAddressWithPassphrase(account.GetKeyPair().GenerateAddress(), "none")
	assert.NotEqual(t, account, account2)
	assert.NotEqual(t, nil, err1)
	assert.Equal(t, nil, err)
	assert.Equal(t, account, account1)
}

func TestAccountManager_GetAccountByUnfoundAddress(t *testing.T) {
	am := NewAccountManager(nil)
	account := client.NewAccount()
	assert.Nil(t, am.GetAccountByAddress(account.GetKeyPair().GenerateAddress()))
}

func TestAccountManager_GetAccountByAddressNilInput(t *testing.T) {
	am := NewAccountManager(nil)
	assert.Nil(t, am.GetAccountByAddress(client.NewAddress("")))
}

func TestAccountManager_GetKeyPairByAddress(t *testing.T) {
	am := NewAccountManager(nil)
	account := client.NewAccount()
	am.Accounts = append(am.Accounts, account)
	assert.Equal(t, account.GetKeyPair(), am.GetKeyPairByAddress(account.GetKeyPair().GenerateAddress()))
}

func TestAccountManager_GetKeyPairByUnfoundAddress(t *testing.T) {
	am := NewAccountManager(nil)
	account := client.NewAccount()
	assert.Nil(t, am.GetKeyPairByAddress(account.GetKeyPair().GenerateAddress()))
}

func TestAccountManager_GetKeyPairByAddressNilInput(t *testing.T) {
	am := NewAccountManager(nil)
	assert.Nil(t, am.GetKeyPairByAddress(client.Address{}))
}

func TestNewAccountManager_UnlockTimer(t *testing.T) {
	fl := storage.NewFileLoader(strings.Replace(GetAccountFilePath(), "accounts", "accounts_test", -1))
	am := NewAccountManager(fl)
	err1 := am.LoadFromFile()
	if err1 != nil {
		fmt.Println(err1.Error())
	}
	passBytes, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	account := client.NewAccount()
	am.AddAccount(account)
	am.PassPhrase = passBytes
	am.Locked = true
	am.SaveAccountToFile()

	am.SetUnlockTimer(10 * time.Second)
	assert.Equal(t, false, am.Locked)
	time.Sleep(3 * time.Second)
	am.mutex.Lock()
	assert.Equal(t, false, am.Locked)
	am.mutex.Unlock()
	time.Sleep(9 * time.Second)
	fl2 := storage.NewFileLoader(strings.Replace(GetAccountFilePath(), "accounts", "accounts_test", -1))
	am2 := NewAccountManager(fl2)
	am2.LoadFromFile()
	assert.Equal(t, true, am2.Locked)
}

func TestAccountManager_Proto(t *testing.T) {
	am := NewAccountManager(nil)
	account := client.NewAccount()
	am.AddAccount(account)
	account = client.NewAccount()
	am.AddAccount(account)

	rawBytes, err := proto.Marshal(am.ToProto())
	assert.Nil(t, err)
	amProto := &logicpb.AccountManager{}
	err = proto.Unmarshal(rawBytes, amProto)
	assert.Nil(t, err)
	am1 := &AccountManager{}
	am1.FromProto(amProto)
	assert.Equal(t, am.Accounts, am1.Accounts)
	assert.Equal(t, am.PassPhrase, am1.PassPhrase)
	assert.Equal(t, am.Locked, am1.Locked)
}
