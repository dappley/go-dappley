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

package logic

import (
	"fmt"
	"github.com/dappley/go-dappley/core/transaction"
	"os"
	"testing"

	"github.com/dappley/go-dappley/logic/transactionpool"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const InvalidAddress = "Invalid Address"

func TestMain(m *testing.M) {
	cleanUpDatabase()
	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	cleanUpDatabase()
	os.Exit(retCode)
}

func TestCreateAccount(t *testing.T) {
	acc, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)
	_, err = account.IsValidPubKey(acc.GetKeyPair().GetPublicKey())
	assert.Nil(t, err)
	cleanUpDatabase()
}

func TestCreateAccountWithPassphrase(t *testing.T) {
	acc, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)
	_, err = account.IsValidPubKey(acc.GetKeyPair().GetPublicKey())
	assert.Nil(t, err)
	cleanUpDatabase()
}

func TestCreateAccountWithPassphraseMismatch(t *testing.T) {
	_, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)

	_, err = CreateAccountWithPassphrase("wrong_password", GetTestAccountPath())
	assert.Error(t, err)
	cleanUpDatabase()
}

func TestCreateBlockchain(t *testing.T) {

	store := storage.NewRamStorage()
	defer store.Close()

	//create a account address
	addr := account.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")

	//create a blockchain
	_, err := CreateBlockchain(addr, store, nil, transactionpool.NewTransactionPool(nil, 128), nil, 1000000)
	assert.Nil(t, err)
}

func TestLoopCreateBlockchain(t *testing.T) {

	store := storage.NewRamStorage()
	defer store.Close()

	//create a account address

	err := ErrInvalidAddress
	//create a blockchain loop
	for i := 0; i < 2000; i++ {
		err = nil
		account := account.NewAccount()
		if !account.IsValid() {
			fmt.Println(i, account.GetAddress())
			err = ErrInvalidAddress
			break
		}
	}
	assert.Nil(t, err)
}

//create a blockchain with invalid address
func TestCreateBlockchainWithInvalidAddress(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	//create a blockchain with an invalid address
	bc, err := CreateBlockchain(account.NewAddress(InvalidAddress), store, nil, transactionpool.NewTransactionPool(nil, 128), nil, 1000000)
	assert.Equal(t, ErrInvalidAddress, err)
	assert.Nil(t, bc)
}

func TestGetBalance(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	//create a account address
	addr := account.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil, transactionpool.NewTransactionPool(nil, 128), nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10000000 after creating a blockchain
	balance, err := GetBalance(addr, bc)
	assert.Nil(t, err)
	assert.Equal(t, transaction.Subsidy, balance)
}

func TestGetBalanceWithInvalidAddress(t *testing.T) {

	store := storage.NewRamStorage()
	defer store.Close()

	//create a account address
	addr := account.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil, transactionpool.NewTransactionPool(nil, 128), nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10000000 after creating a blockchain
	balance1, err := GetBalance(account.NewAddress("dG6HhzSdA5m7KqvJNszVSf8i5f4neAteSs"), bc)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance1)

	balance2, err := GetBalance(account.NewAddress("dG6HhzSdA5m7KqvJNszVSf8i5f4neAtfSs"), bc)
	assert.Equal(t, ErrInvalidAddress, err)
	assert.Equal(t, common.NewAmount(0), balance2)
}

func TestGetAllAddresses(t *testing.T) {
	cleanUpDatabase()

	store := storage.NewRamStorage()
	defer store.Close()

	expectedRes := []account.Address{}
	//create a account address
	account, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.NotEmpty(t, account)
	addr := account.GetAddress()

	expectedRes = append(expectedRes, addr)

	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil, transactionpool.NewTransactionPool(nil, 128), nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//create 10 more addresses
	for i := 0; i < 2; i++ {
		//create a account address
		account, err = CreateAccountWithPassphrase("test", GetTestAccountPath())
		addr = account.GetAddress()
		assert.NotEmpty(t, addr)
		assert.Nil(t, err)
		expectedRes = append(expectedRes, addr)
	}

	//get all addresses
	addrs, err := GetAllAddressesByPath(GetTestAccountPath())
	assert.Nil(t, err)

	//the length should be equal
	assert.Equal(t, len(expectedRes), len(addrs))
	assert.ElementsMatch(t, expectedRes, addrs)
	cleanUpDatabase()
}

func TestIsAccountEmptyAccount(t *testing.T) {
	account1, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.NotEmpty(t, account1)
	assert.Nil(t, err)
	empty, err := IsAccountEmpty(GetTestAccountPath())
	assert.Nil(t, err)
	assert.Equal(t, false, empty)
	cleanUpDatabase()
	empty, err = IsAccountEmpty(GetTestAccountPath())
	assert.Nil(t, err)
	assert.Equal(t, true, empty)
}

func TestDeleteInvalidAccount(t *testing.T) {
	//create accounts address
	account1, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.NotEmpty(t, account1)
	addr1 := account1.GetAddress()

	addressList := []account.Address{addr1}

	list, err := GetAllAddressesByPath(GetTestAccountPath())
	assert.Nil(t, err)
	assert.ElementsMatch(t, list, addressList)
	cleanUpDatabase()
}

func TestIsAccountLocked(t *testing.T) {
	_, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)

	status, err := IsAccountLocked(GetTestAccountPath())
	assert.Nil(t, err)
	assert.True(t, status)
	cleanUpDatabase()
}

func TestNilSetLockAccount(t *testing.T) {
	assert.Nil(t, SetLockAccount(GetTestAccountPath()))
}

func TestSetLockAccount(t *testing.T) {
	_, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)

	assert.Nil(t, SetLockAccount(GetTestAccountPath()))
	status, err := IsAccountLocked(GetTestAccountPath())
	assert.Nil(t, err)
	assert.True(t, status)

	cleanUpDatabase()

	status, err = IsAccountLocked(GetTestAccountPath())
	assert.Nil(t, err)
	assert.False(t, status)
}

func TestSetUnLockAccount(t *testing.T) {
	_, err := CreateAccountWithPassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)

	assert.Nil(t, SetUnLockAccount(GetTestAccountPath()))
	status, err := IsAccountLocked(GetTestAccountPath())
	assert.Nil(t, err)
	assert.False(t, status)
	cleanUpDatabase()
}

func cleanUpDatabase() {
	RemoveAccountTestFile()
}
