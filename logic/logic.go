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

	"github.com/dappley/go-dappley/common"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/network"
)

var (
	ErrInvalidAmount        = errors.New("ERROR: Amount is invalid (must be > 0)")
	ErrInvalidAddress       = errors.New("ERROR: Address is invalid")
	ErrInvalidSenderAddress = errors.New("ERROR: Sender address is invalid")
	ErrInvalidRcverAddress  = errors.New("ERROR: Receiver address is invalid")
)

//create a blockchain
func CreateBlockchain(address core.Address, db storage.Storage, consensus core.Consensus) (*core.Blockchain, error) {
	if !address.ValidateAddress() {
		return nil, ErrInvalidAddress
	}

	bc := core.CreateBlockchain(address, db, consensus)

	return bc, nil
}

//create a wallet
func CreateWallet() (*client.Wallet, error) {
	fl := storage.NewFileLoader(client.GetWalletFilePath())
	wm := client.NewWalletManager(fl)
	err := wm.LoadFromFile()
	wallet := client.NewWallet()
	wm.AddWallet(wallet)
	wm.SaveWalletToFile()

	return wallet, err
}

//create a wallet with passphrase
func CreateWalletWithpassphrase(passphrase string) (*client.Wallet, error) {
	fl := storage.NewFileLoader(client.GetWalletFilePath())
	wm := client.NewWalletManager(fl)
	err := wm.LoadFromFile()
	wallet := client.NewWalletWithPassphrase(passphrase)
	wm.AddWallet(wallet)
	wm.SaveWalletToFile()
	wm.LoadFromFile()

	return wallet, err
}

//get balance
func GetBalance(address core.Address, db storage.Storage) (*common.Amount, error) {
	pubKeyHash, valid := address.GetPubKeyHash()
	if valid == false {
		return common.NewAmount(0), ErrInvalidAddress
	}

	balance := common.NewAmount(0)
	utxoIndex := core.LoadUTXOIndex(db)
	utxos := utxoIndex.GetUTXOsByPubKey(pubKeyHash)
	for _, out := range utxos {
		balance = balance.Add(out.Value)
	}

	return balance, nil
}

//get all addresses
func GetAllAddresses() ([]core.Address, error) {
	fl := storage.NewFileLoader(client.GetWalletFilePath())
	wm := client.NewWalletManager(fl)
	err := wm.LoadFromFile()
	if err != nil {
		return nil, err
	}

	addresses := wm.GetAddresses()

	return addresses, err
}

func Send(senderWallet *client.Wallet, to core.Address, amount *common.Amount, tip uint64, bc *core.Blockchain, node *network.Node) error {
	if !senderWallet.GetAddress().ValidateAddress() {
		return ErrInvalidSenderAddress
	}
	if !to.ValidateAddress() {
		return ErrInvalidRcverAddress
	}
	if amount.Validate() != nil || amount.IsZero() {
		return ErrInvalidAmount
	}

	tx, err := core.NewUTXOTransaction(bc.GetDb(), senderWallet.GetAddress(), to, amount, *senderWallet.GetKeyPair(), bc, tip)
	bc.GetTxPool().ConditionalAdd(tx)
	node.TxBroadcast(&tx)
	if err != nil {
		return err
	}

	return err
}

//add balance
func AddBalance(address core.Address, amount *common.Amount, bc *core.Blockchain) error {
	if !address.ValidateAddress() {
		return ErrInvalidAddress
	}

	if amount.Validate() != nil || amount.IsZero() {
		return ErrInvalidAmount
	}

	tx, err := core.NewUTXOTransactionforAddBalance(address, amount)

	if err != nil {
		return err
	}

	bc.GetTxPool().StructPush(tx)

	return err

}

