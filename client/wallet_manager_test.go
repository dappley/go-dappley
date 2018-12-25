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
package client

import (
	"errors"
	"fmt"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/storage/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"testing"
	"time"
)

func TestWalletManager_LoadFromFileExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().ReadFromFile()

	wm := NewWalletManager(mockStorage)
	wm.LoadFromFile()

}

func TestWalletManager_LoadFromFileNotExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)

	gomock.InOrder(
		mockStorage.EXPECT().ReadFromFile().Return(nil, errors.New("err")),
		mockStorage.EXPECT().SaveToFile(gomock.Any()),
		mockStorage.EXPECT().ReadFromFile(),
	)

	wm := NewWalletManager(mockStorage)
	wm.LoadFromFile()
}

func TestWalletManager_SaveWalletToFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().SaveToFile(gomock.Any())
	wm := NewWalletManager(mockStorage)
	wm.SaveWalletToFile()

}

func TestWalletManager_SaveWalletToFile_with_passphrase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().SaveToFile(gomock.Any())
	wm := NewWalletManager(mockStorage)
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	wm.SaveWalletToFile()

}

func TestWalletManager_AddWallet(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.AddWallet(wallet)

	assert.Equal(t, wallet, wm.Wallets[0])
}

func TestWallet_GetAddresses(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	assert.Equal(t, wallet.GetAddresses(), wm.GetAddresses())
}

func TestWallet_GetAddressesNoWallet(t *testing.T) {
	wm := NewWalletManager(nil)
	assert.Equal(t, []core.Address(nil), wm.GetAddresses())
}

func TestWalletManager_GetWalletByAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	assert.Equal(t, wallet, wm.GetWalletByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetWalletByAddress_withPassphrase(t *testing.T) {
	wm := NewWalletManager(nil)
	passPhrase, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.Equal(t, nil, err)
	wm.PassPhrase = passPhrase
	wm.Locked = true
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	wallet1, err := wm.GetWalletByAddressWithPassphrase(wallet.GetAddress(), "password")
	wallet2, err1 := wm.GetWalletByAddressWithPassphrase(wallet.GetAddress(), "none")
	assert.NotEqual(t, wallet, wallet2)
	assert.NotEqual(t, nil, err1)
	assert.Equal(t, nil, err)
	assert.Equal(t, wallet, wallet1)
}

func TestWalletManager_GetWalletByUnfoundAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	assert.Nil(t, wm.GetWalletByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetWalletByAddressNilInput(t *testing.T) {
	wm := NewWalletManager(nil)
	assert.Nil(t, wm.GetWalletByAddress(core.Address{}))
}

func TestWalletManager_GetKeyPairByAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	assert.Equal(t, wallet.Key, wm.GetKeyPairByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetKeyPairByUnfoundAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	assert.Nil(t, wm.GetKeyPairByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetKeyPairByAddressNilInput(t *testing.T) {
	wm := NewWalletManager(nil)
	assert.Nil(t, wm.GetKeyPairByAddress(core.Address{}))
}

func TestNewWalletManager_UnlockTimer(t *testing.T) {
	fl := storage.NewFileLoader(strings.Replace(GetWalletFilePath(), "wallets", "wallets_test", -1))
	wm := NewWalletManager(fl)
	err1 := wm.LoadFromFile()
	if err1 != nil {
		fmt.Println(err1.Error())
	}
	passBytes, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	wallet := NewWallet()
	wm.AddWallet(wallet)
	wm.PassPhrase = passBytes
	wm.Locked = true
	wm.SaveWalletToFile()

	wm.SetUnlockTimer(10 * time.Second)
	assert.Equal(t, false, wm.Locked)
	time.Sleep(3 * time.Second)
	wm.mutex.Lock()
	assert.Equal(t, false, wm.Locked)
	wm.mutex.Unlock()
	time.Sleep(9 * time.Second)
	fl2 := storage.NewFileLoader(strings.Replace(GetWalletFilePath(), "wallets", "wallets_test", -1))
	wm2 := NewWalletManager(fl2)
	wm2.LoadFromFile()
	assert.Equal(t, true, wm2.Locked)
}
