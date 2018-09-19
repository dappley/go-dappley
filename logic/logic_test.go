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
	"errors"
	"os"
	"testing"

	"github.com/dappley/go-dappley/common"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
	logger "github.com/sirupsen/logrus"
	"reflect"
)

const InvalidAddress = "Invalid Address"

func TestMain(m *testing.M) {
	setup()
	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	teardown()
	os.Exit(retCode)
}

func TestCreateWallet(t *testing.T) {
	wallet, err := CreateWallet()
	assert.Nil(t, err)
	expectedLength := 34
	if hash, _ := core.HashPubKey(wallet.GetKeyPair().PublicKey);hash[0] < 10{
		expectedLength = 33
	}
	assert.Equal(t, expectedLength, len(wallet.Addresses[0].Address))
}

func TestCreateWalletWithPassphrase(t *testing.T) {
	wallet, err := CreateWalletWithpassphrase("passpass")
	assert.Nil(t, err)
	expectedLength := 34
	if hash, _ := core.HashPubKey(wallet.GetKeyPair().PublicKey);hash[0] < 10{
		expectedLength = 33
	}
	assert.Equal(t, expectedLength, len(wallet.Addresses[0].Address))

}

func TestCreateBlockchain(t *testing.T) {

	store := storage.NewRamStorage()
	defer store.Close()

	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}

	//create a blockchain
	_, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
}

//create a blockchain with invalid address
func TestCreateBlockchainWithInvalidAddress(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	//create a blockchain with an invalid address
	bc, err := CreateBlockchain(core.NewAddress(InvalidAddress), store, nil)
	assert.Equal(t, ErrInvalidAddress, err)
	assert.Nil(t, bc)
}

func TestGetBalance(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance, err := GetBalance(addr, store)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(10), balance)
}

func TestGetBalanceWithInvalidAddress(t *testing.T) {

	store := storage.NewRamStorage()
	defer store.Close()

	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(core.NewAddress("1AUrNJCRM5X5fDdmm3E3yjCrXQMLvDj9tb"), store)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance1)

	balance2, err := GetBalance(core.NewAddress("1AUrNJCRM5X5fDdmm3E3yjCrXQMLwfwfww"), store)
	assert.Equal(t, errors.New("ERROR: Address is invalid"), err)
	assert.Equal(t, common.NewAmount(0), balance2)
}

func TestGetAllAddresses(t *testing.T) {
	setup()

	store := storage.NewRamStorage()
	defer store.Close()

	expected_res := []core.Address{}
	//create a wallet address
	wallet, err := CreateWallet()
	assert.NotEmpty(t, wallet)
	addr := wallet.GetAddress()

	expected_res = append(expected_res, addr)

	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//create 10 more addresses
	for i := 0; i < 2; i++ {
		//create a wallet address
		wallet, err = CreateWallet()
		addr = wallet.GetAddress()
		assert.NotEmpty(t, addr)
		assert.Nil(t, err)
		expected_res = append(expected_res, addr)
	}

	//get all addresses
	addrs, err := GetAllAddresses()
	assert.Nil(t, err)

	//the length should be equal
	assert.Equal(t, len(expected_res), len(addrs))
	assert.ElementsMatch(t, expected_res, addrs)
	teardown()
}

func TestDeleteInvalidWallet(t *testing.T) {
	//setup: clean up database and files
	setup()
	//create wallets address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	addressList := []core.Address{addr1}

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.ElementsMatch(t, list, addressList)

	//teardown :clean up database amd files
	teardown()
}

func TestCompare(t *testing.T) {
	bc1 := core.GenerateMockBlockchain(5)
	bc2 := bc1
	assert.True(t, compareTwoBlockchains(bc1, bc2))
	bc3 := core.GenerateMockBlockchain(5)
	assert.False(t, compareTwoBlockchains(bc1, bc3))
}

func compareTwoBlockchains(bc1, bc2 *core.Blockchain) bool {
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

func setup() {
	cleanUpDatabase()
}

func teardown() {
	cleanUpDatabase()
}

func cleanUpDatabase() {
	client.RemoveWalletFile()
}
