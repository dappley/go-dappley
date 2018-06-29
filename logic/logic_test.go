// Copyright (C) 2018 go-dappworks authors
//
// This file is part of the go-dappworks library.
//
// the go-dappworks library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappworks library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappworks library.  If not, see <http://www.gnu.org/licenses/>.
//

package logic

import (
	"os"
	"testing"

	"github.com/dappworks/go-dappworks/client"
	"github.com/dappworks/go-dappworks/storage"
	"github.com/stretchr/testify/assert"
)

func TestCreateBlockchain(t *testing.T) {

	//setup: clean up database and files
	setup()

	//create a wallet address
	addr, err := CreateWallet()
	assert.NotEmpty(t, addr)

	//create a blockchain
	b, err := CreateBlockchain(addr)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//teardown :clean up database amd files
	teardown()
}

func TestGetBalance(t *testing.T) {
	//setup: clean up database and files
	setup()

	//create a wallet address
	addr, err := CreateWallet()
	assert.NotEmpty(t, addr)

	//create a blockchain
	b, err := CreateBlockchain(addr)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance, err := GetBalance(addr)
	assert.Nil(t, err)
	assert.Equal(t, balance, 10)

	//teardown :clean up database amd files
	teardown()
}

func TestGetAllAddresses(t *testing.T) {

	//setup: clean up database and files
	setup()

	expected_res := []string{}
	//create a wallet address
	addr, err := CreateWallet()
	assert.NotEmpty(t, addr)
	expected_res = append(expected_res, addr)

	//create a blockchain
	b, err := CreateBlockchain(addr)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//create 10 more addresses
	for i := 0; i < 10; i++ {
		//create a wallet address
		addr, err = CreateWallet()
		assert.NotEmpty(t, addr)
		assert.Nil(t, err)
		expected_res = append(expected_res, addr)
	}

	//get all addresses
	addrs, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.NotNil(t, addrs)

	//the length should be equal
	assert.Equal(t, len(expected_res), len(addrs))
	assert.ElementsMatch(t, expected_res, addrs)

	//teardown :clean up database amd files
	teardown()
}

func TestSend(t *testing.T) {
	//setup: clean up database and files
	setup()
	mineAward := int(10)
	transferAmount := int(5)

	//create a wallet address
	addr1, err := CreateWallet()
	assert.NotEmpty(t, addr1)

	//create a blockchain
	b, err := CreateBlockchain(addr1)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance1 should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineAward)

	//Create a second wallet
	addr2, err := CreateWallet()
	assert.NotEmpty(t, addr2)
	assert.Nil(t, err)

	//The balance1 should be 0
	balance2, err := GetBalance(addr2)
	assert.Nil(t, err)
	assert.Equal(t, balance2, 0)

	//Send 5 coins from addr1 to addr2
	err = Send(addr1, addr2, transferAmount)
	assert.Nil(t, err)

	//the balance1 of the first wallet should be 10-5+10(mining new block)=15
	balance1, err = GetBalance(addr1)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineAward-transferAmount+mineAward)

	//the balance1 of the second wallet should be 5
	balance2, err = GetBalance(addr2)
	assert.Nil(t, err)
	assert.Equal(t, balance2, transferAmount)

	//teardown :clean up database amd files
	teardown()
}

func TestDeleteWallets(t *testing.T) {
	//setup: clean up database and files
	setup()

	//create wallets address
	addr1, err := CreateWallet()
	assert.Nil(t, err)
	assert.NotEmpty(t, addr1)

	addr2, err := CreateWallet()
	assert.Nil(t, err)
	assert.NotEmpty(t, addr2)

	addr3, err := CreateWallet()
	assert.Nil(t, err)
	assert.NotEmpty(t, addr3)

	err = DeleteWallets()
	assert.Nil(t, err)

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.Empty(t, list)

	//teardown :clean up database amd files
	teardown()
}

func TestDeleteWallet(t *testing.T) {
	//setup: clean up database and files
	setup()

	//create wallets address
	addr1, err := CreateWallet()
	assert.Nil(t, err)
	assert.NotEmpty(t, addr1)

	addr2, err := CreateWallet()
	assert.Nil(t, err)
	assert.NotEmpty(t, addr2)

	addr3, err := CreateWallet()
	assert.Nil(t, err)
	assert.NotEmpty(t, addr3)

	addressList := [2]string{addr2, addr3}

	err = DeleteWallet(addr1)
	assert.Nil(t, err)

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.Equal(t, list, addressList)

	//teardown :clean up database amd files
	teardown()
}

func setup() {
	cleanUpDatabase()
}

func teardown() {
	cleanUpDatabase()
}

func cleanUpDatabase() {
	os.RemoveAll(storage.DefaultDbFile)
	os.RemoveAll(client.WalletFile)
}
