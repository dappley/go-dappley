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

	"github.com/dappworks/go-dappworks/client"
	"github.com/dappworks/go-dappworks/core"
	"github.com/dappworks/go-dappworks/util"
)

var (
	ErrInvalidAddress       = errors.New("ERROR: Address is invalid")
	ErrInvalidSenderAddress = errors.New("ERROR: Sender address is invalid")
	ErrInvalidRcverAddress  = errors.New("ERROR: Receiver address is invalid")
)

//create a blockchain
func CreateBlockchain(address string) (*core.Blockchain, error) {
	if !core.ValidateAddress(address) {
		return nil, ErrInvalidAddress
	}
	bc := core.CreateBlockchain(address)
	err := bc.DB.Close()
	return bc, err
}

//create a wallet
func CreateWallet() (string, error) {
	wallets, err := client.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()
	return address, err
}

//get balance
func GetBalance(address string) (int, error) {
	if !core.ValidateAddress(address) {
		return 0, ErrInvalidAddress
	}
	bc := core.GetBlockchain(address)
	defer bc.DB.Close()

	balance := 0
	pubKeyHash := util.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := bc.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}
	return balance, nil
}

//get all addresses
func GetAllAddresses() ([]string, error) {
	wallets, err := client.NewWallets()
	if err != nil {
		return nil, err
	}
	addresses := wallets.GetAddresses()

	return addresses, err
}

func Send(from, to string, amount int, tip int64) error {
	if !core.ValidateAddress(from) {
		return ErrInvalidSenderAddress
	}
	if !core.ValidateAddress(to) {
		return ErrInvalidRcverAddress
	}

	bc := core.GetBlockchain(from)
	defer bc.DB.Close()

	wallets, err := client.NewWallets()
	if err != nil {
		return err
	}
	wallet := wallets.GetWallet(from)
	tx, err := core.NewUTXOTransaction(from, to, amount, wallet, bc, tip)
	if err != nil {
		return err
	}
	cbTx := core.NewCoinbaseTX(from, "")
	txs := []*core.Transaction{cbTx, tx}

	//TODO: miner should be separated from the sender
	bc.MineBlock(txs)
	return err
}

//delete wallet

func DeleteWallet(address string) error {
	wallets, err := client.NewWallets()
	if err != nil {
		return err
	}
	err = wallets.DeleteWallet(address)
	if err != nil {
		return err
	}
	wallets.SaveToFile()
	return err
}

func DeleteWallets() error {
	wallets, err := client.NewWallets()
	if err != nil {
		return err
	}
	err = wallets.DeleteWallets()
	if err != nil {
		return err
	}
	wallets.SaveToFile()
	return err
}
