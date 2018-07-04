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
	"errors"
	"os"
	"testing"

	"github.com/dappworks/go-dappworks/client"
	"github.com/dappworks/go-dappworks/storage"
	"github.com/stretchr/testify/assert"
)
const invalidAddress = "Invalid Address"

func TestCreateWallet(t *testing.T) {
	//setup: clean up database and files
	setup()

	addr, _ := CreateWallet()
	assert.NotEmpty(t, addr)
	//teardown :clean up database amd files
	teardown()
}

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

//create a blockchain with invalid address
func TestCreateBlockchainWithInvalidAddress(t *testing.T) {
	//setup: clean up database and files
	setup()


	//create a blockchain with an invalid address
	b, err := CreateBlockchain(invalidAddress)
	assert.Equal(t, err, ErrInvalidAddress)
	assert.Nil(t, b)
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

func TestGetBalanceWithInvildeAddress(t *testing.T) {
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
	balance1, err := GetBalance("1AUrNJCRM5X5fDdmm3E3yjCrXQMLvDj9tb")
	assert.Nil(t, err)
	assert.Equal(t, balance1, 0)

	balance2, err := GetBalance("1AUrNJCRM5X5fDdmm3E3yjCrXQMLwfwfww")
	assert.Equal(t, errors.New("ERROR: Address is invalid"), err)
	assert.Equal(t, balance2, 0)

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

//test send
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
	assert.Equal(t,mineAward, balance1)

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
	assert.Equal(t,mineAward-transferAmount+mineAward, balance1)

	//the balance1 of the second wallet should be 5
	balance2, err = GetBalance(addr2)
	assert.Nil(t, err)
	assert.Equal(t, transferAmount, balance2)

	//teardown :clean up database amd files
	teardown()
}

func TestDeleteWallets(t *testing.T) {
	//setup: clean up database and files
	setup()

	//create wallets address
	addr1, err := CreateWallet()
	assert.NotEmpty(t, addr1)

	addr2, err := CreateWallet()
	assert.NotEmpty(t, addr2)

	addr3, err := CreateWallet()
	assert.NotEmpty(t, addr3)

	err = DeleteWallets()
	assert.Nil(t, err)

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.Empty(t, list)

	teardown()
}

//test send to invalid address
func TestSendToInvalidAddress(t *testing.T) {
	//setup: clean up database and files
	setup()

	//this is internally set. Dont modify
	mineAward := int(10)
	//Transfer ammount
	transferAmount := int(25)

	//create a wallet address
	addr1, err := CreateWallet()
	assert.NotEmpty(t, addr1)

	//create a blockchain
	b, err := CreateBlockchain(addr1)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineAward)

	//Send 5 coins from addr1 to an invalid address
	err = Send(addr1, invalidAddress, transferAmount)
	assert.NotNil(t, err)

	//the balance of the first wallet should be still be 10
	balance1, err = GetBalance(addr1)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineAward)
	//teardown :clean up database amd files
	teardown()
}

func TestDeleteWallet(t *testing.T) {
	//setup: clean up database and files
	setup()

	//create wallets address
	addr1, err := CreateWallet()
	assert.NotEmpty(t, addr1)

	addr2, err := CreateWallet()
	assert.NotEmpty(t, addr2)

	addr3, err := CreateWallet()
	assert.NotEmpty(t, addr3)

	addressList := []string{addr2, addr3}

	err = DeleteWallet(addr1)
	assert.Nil(t, err)

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.ElementsMatch(t, list, addressList)

	//teardown :clean up database amd files
	teardown()
}

func TestDeleteInvildeWallet(t *testing.T) {
	//setup: clean up database and files
	setup()

	//create wallets address
	addr1, err := CreateWallet()
	assert.NotEmpty(t, addr1)

	addressList := []string{addr1}

	println(addr1)

	err = DeleteWallet("1AUrNJCRM5X5fDdmm3E3yjCrXQMLvDj9tb")
	assert.Equal(t, errors.New("wallet is not exist"), err)

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.ElementsMatch(t, list, addressList)

	//teardown :clean up database amd files
	teardown()
}

//insufficient fund
func TestSendInefficientBalance(t *testing.T) {
	//setup: clean up database and files
	setup()

	//this is internally set. Dont modify
	mineAward := int(10)
	//Transfer ammount is larger than the balance
	transferAmount := int(25)

	//create a wallet address
	addr1, err := CreateWallet()
	assert.NotEmpty(t, addr1)

	//create a blockchain
	b, err := CreateBlockchain(addr1)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineAward)

	//Create a second wallet
	addr2, err := CreateWallet()
	assert.NotEmpty(t, addr2)
	assert.Nil(t, err)

	//The balance should be 0
	balance2, err := GetBalance(addr2)
	assert.Nil(t, err)
	assert.Equal(t, balance2, 0)

	//Send 5 coins from addr1 to addr2
	err = Send(addr1, addr2, transferAmount)
	assert.NotNil(t, err)

	//the balance of the first wallet should be still be 10
	balance1, err = GetBalance(addr1)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineAward)

	//the balance of the second wallet should be 0
	balance2, err = GetBalance(addr2)
	assert.Nil(t, err)
	assert.Equal(t, balance2, 0)

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
