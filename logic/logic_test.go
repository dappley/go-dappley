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
	"os"
	"reflect"
	"testing"

	"github.com/dappley/go-dappley/core/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic/account_logic"
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
	account, err := CreateAccount(GetTestAccountPath(), "test")
	assert.Nil(t, err)
	pubKeyHash, ok := client.GeneratePubKeyHashByAddress(account.GetKeyPair().GenerateAddress())
	assert.Equal(t, true, ok)
	accountPubKeyHash, err := client.NewUserPubKeyHash(account.GetKeyPair().PublicKey)
	assert.Nil(t, err)
	assert.Equal(t, pubKeyHash, accountPubKeyHash)
}

func TestCreateAccountWithPassphrase(t *testing.T) {
	account, err := CreateAccountWithpassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)
	pubKeyHash, ok := client.GeneratePubKeyHashByAddress(account.GetKeyPair().GenerateAddress())
	assert.Equal(t, true, ok)
	accountPubKeyHash, err := client.NewUserPubKeyHash(account.GetKeyPair().PublicKey)
	assert.Nil(t, err)
	assert.Equal(t, pubKeyHash, accountPubKeyHash)
}

func TestCreateAccountWithPassphraseMismatch(t *testing.T) {
	_, err := CreateAccountWithpassphrase("test", GetTestAccountPath())
	assert.Nil(t, err)

	_, err = CreateAccountWithpassphrase("wrong_password", GetTestAccountPath())
	assert.Error(t, err)
}

func TestCreateBlockchain(t *testing.T) {

	store := storage.NewRamStorage()
	defer store.Close()

	//create a account address
	addr := client.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")

	//create a blockchain
	_, err := CreateBlockchain(addr, store, nil, 128, nil, 1000000)
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
		account := client.NewAccount()
		addr := account.GetKeyPair().GenerateAddress()
		if !addr.IsValid() {
			fmt.Println(i, addr)
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
	bc, err := CreateBlockchain(client.NewAddress(InvalidAddress), store, nil, 128, nil, 1000000)
	assert.Equal(t, ErrInvalidAddress, err)
	assert.Nil(t, bc)
}

func TestGetBalance(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	//create a account address
	addr := client.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil, 128, nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10000000 after creating a blockchain
	balance, err := GetBalance(addr, bc)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(10000000), balance)
}

func TestGetBalanceWithInvalidAddress(t *testing.T) {

	store := storage.NewRamStorage()
	defer store.Close()

	//create a account address
	addr := client.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil, 128, nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10000000 after creating a blockchain
	balance1, err := GetBalance(client.NewAddress("dG6HhzSdA5m7KqvJNszVSf8i5f4neAteSs"), bc)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance1)

	balance2, err := GetBalance(client.NewAddress("dG6HhzSdA5m7KqvJNszVSf8i5f4neAtfSs"), bc)
	assert.Equal(t, ErrInvalidAddress, err)
	assert.Equal(t, common.NewAmount(0), balance2)
}

func TestGetAllAddresses(t *testing.T) {
	cleanUpDatabase()

	store := storage.NewRamStorage()
	defer store.Close()

	expectedRes := []client.Address{}
	//create a account address
	account, err := CreateAccount(GetTestAccountPath(), "test")
	assert.NotEmpty(t, account)
	addr := account.GetKeyPair().GenerateAddress()

	expectedRes = append(expectedRes, addr)

	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil, 128, nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//create 10 more addresses
	for i := 0; i < 2; i++ {
		//create a account address
		account, err = CreateAccount(GetTestAccountPath(), "test")
		addr = account.GetKeyPair().GenerateAddress()
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
	account1, err := CreateAccount(GetTestAccountPath(), "test")
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
	account1, err := CreateAccount(GetTestAccountPath(), "test")
	assert.NotEmpty(t, account1)
	addr1 := account1.GetKeyPair().GenerateAddress()

	addressList := []client.Address{addr1}

	list, err := GetAllAddressesByPath(GetTestAccountPath())
	assert.Nil(t, err)
	assert.ElementsMatch(t, list, addressList)
}

func TestIsAccountLocked(t *testing.T) {
	_, err := CreateAccount(GetTestAccountPath(), "test")
	assert.Nil(t, err)

	status, err := IsAccountLocked(GetTestAccountPath())
	assert.Nil(t, err)
	assert.True(t, status)
}

func TestNilSetLockAccount(t *testing.T) {
	assert.Nil(t, SetLockAccount(GetTestAccountPath()))
}

func TestSetLockAccount(t *testing.T) {
	_, err := CreateAccount(GetTestAccountPath(), "test")
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
	_, err := CreateAccount(GetTestAccountPath(), "test")
	assert.Nil(t, err)

	assert.Nil(t, SetUnLockAccount(GetTestAccountPath()))
	status, err := IsAccountLocked(GetTestAccountPath())
	assert.Nil(t, err)
	assert.False(t, status)
}

func isSameBlockChain(bc1, bc2 *core.Blockchain) bool {
	if bc1 == nil || bc2 == nil {
		return false
	}

	bci1 := bc1.Iterator()
	bci2 := bc2.Iterator()
	if bc1.GetMaxHeight() != bc2.GetMaxHeight() {
		return false
	}

loop:
	for {
		blk1, _ := bci1.Next()
		blk2, _ := bci2.Next()
		if blk1 == nil || blk2 == nil {
			break loop
		}
		if !reflect.DeepEqual(blk1.GetHash(), blk2.GetHash()) {
			return false
		}
	}
	return true
}

func cleanUpDatabase() {
	account_logic.RemoveAccountFile()
}
