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
	"github.com/dappley/go-dappley/contract"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const unlockduration = 300 * time.Second

var minerPrivateKey string
var (
	ErrInvalidAmount        = errors.New("ERROR: Amount is invalid (must be > 0)")
	ErrInvalidAddress       = errors.New("ERROR: Address is invalid")
	ErrInvalidSenderAddress = errors.New("ERROR: Sender address is invalid")
	ErrInvalidRcverAddress  = errors.New("ERROR: Receiver address is invalid")
	ErrPasswordNotMatch     = errors.New("ERROR: Password not correct")
	ErrPathEmpty            = errors.New("ERROR: Path empty")
	ErrPasswordEmpty        = errors.New("ERROR: Password empty")
)

//create a blockchain
func CreateBlockchain(address core.Address, db storage.Storage, consensus core.Consensus, transactionPoolLimit uint32, scManager *sc.V8EngineManager) (*core.Blockchain, error) {
	if !address.ValidateAddress() {
		return nil, ErrInvalidAddress
	}

	bc := core.CreateBlockchain(address, db, consensus, transactionPoolLimit, scManager)

	return bc, nil
}

//create a wallet from path
func CreateWallet(path string, password string) (*client.Wallet, error) {
	if len(path) == 0 {
		return nil, ErrPathEmpty
	}

	if len(password) == 0 {
		return nil, ErrPasswordEmpty
	}

	fl := storage.NewFileLoader(path)
	wm := client.NewWalletManager(fl)
	passBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	wm.PassPhrase = passBytes
	wm.Locked = true
	err = wm.LoadFromFile()
	wallet := client.NewWallet()
	wm.AddWallet(wallet)
	wm.SaveWalletToFile()

	return wallet, err
}

//get wallet
func GetWallet() (*client.Wallet, error) {
	wm, err := GetWalletManager(client.GetWalletFilePath())
	empty, err := wm.IsFileEmpty()
	if empty {
		return nil, nil
	}
	if len(wm.Wallets) > 0 {
		return wm.Wallets[0], err
	} else {
		return nil, err
	}
}

//Get lock flag
func IsWalletLocked() (bool, error) {
	wm, err := GetWalletManager(client.GetWalletFilePath())
	return wm.Locked, err
}

//Tell if the file empty or not exist
func IsWalletEmpty() (bool, error) {
	if client.Exists(client.GetWalletFilePath()) {
		wm, _ := GetWalletManager(client.GetWalletFilePath())
		if len(wm.Wallets) == 0 {
			return true, nil
		} else {
			return wm.IsFileEmpty()
		}
	} else {
		return true, nil
	}
}

//Set lock flag
func SetLockWallet() error {
	wm, err1 := GetWalletManager(client.GetWalletFilePath())
	empty, err2 := wm.IsFileEmpty()
	if empty {
		return nil
	}
	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	} else {
		wm.Locked = true
		wm.SaveWalletToFile()
		return nil
	}
}

//Set unlock and timer
func SetUnLockWallet() error {
	wm, err := GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		return err
	} else {
		wm.SetUnlockTimer(unlockduration)
		return nil
	}
}

//create a wallet with passphrase
func CreateWalletWithpassphrase(password string) (*client.Wallet, error) {
	wm, err := GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		return nil, err
	}

	if len(wm.Wallets) > 0 && wm.PassPhrase != nil {
		err = bcrypt.CompareHashAndPassword(wm.PassPhrase, []byte(password))
		if err != nil {
			return nil, ErrPasswordNotMatch
		}
		wallet := client.NewWallet()
		wm.AddWallet(wallet)
		wm.SaveWalletToFile()
		return wallet, err

	} else {
		passBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		wm.PassPhrase = passBytes
		logger.Info("Wallet password set!")
		wallet := client.NewWallet()
		wm.AddWallet(wallet)
		wm.Locked = true
		wm.SaveWalletToFile()
		return wallet, err
	}
}

//create a wallet
func AddWallet() (*client.Wallet, error) {
	wm, err := GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		return nil, err
	}

	wallet := client.NewWallet()
	if len(wm.Wallets) == 0 {
		wm.Locked = true
	}
	wm.AddWallet(wallet)
	wm.SaveWalletToFile()
	return wallet, err
}

//Get duration
func GetUnlockDuration() time.Duration {
	return unlockduration
}

//get balance
func GetBalance(address core.Address, db storage.Storage) (*common.Amount, error) {
	pubKeyHash, valid := address.GetPubKeyHash()
	if valid == false {
		return common.NewAmount(0), ErrInvalidAddress
	}

	balance := common.NewAmount(0)
	utxoIndex := core.LoadUTXOIndex(db)
	utxos := utxoIndex.GetAllUTXOsByPubKeyHash(pubKeyHash)
	for _, out := range utxos {
		balance = balance.Add(out.Value)
	}

	return balance, nil
}

func Send(senderWallet *client.Wallet, to core.Address, amount *common.Amount, tip uint64, contract string, bc *core.Blockchain, node *network.Node) ([]byte, error) {
	if !senderWallet.GetAddress().ValidateAddress() {
		return nil, ErrInvalidSenderAddress
	}

	//Contract deployment transaction does not need to validate to address
	if !to.ValidateAddress() && contract == "" {
		return nil, ErrInvalidRcverAddress
	}
	if amount.Validate() != nil || amount.IsZero() {
		return nil, ErrInvalidAmount
	}

	pubKeyHash, _ := core.NewUserPubKeyHash(senderWallet.Key.PublicKey)
	utxos, err := core.LoadUTXOIndex(bc.GetDb()).GetUTXOsByAmount(pubKeyHash.GetPubKeyHash(), amount)
	if err != nil {
		return nil, err
	}

	tx, err := core.NewUTXOTransaction(utxos, senderWallet.GetAddress(), to, amount, *senderWallet.GetKeyPair(), common.NewAmount(tip), contract)
	bc.GetTxPool().Push(tx)
	node.TxBroadcast(&tx)

	contractAddr := tx.GetContractAddress()
	if contractAddr.String() != "" {
		if to.String() == contractAddr.String(){
			logger.WithFields(logger.Fields{
				"contractAddr": contractAddr.String(),
				"data"	  	  : contract,
			}).Info("Smart Contract Invoke Transaction Sent Successful!")
		}else{
			logger.WithFields(logger.Fields{
				"contractAddr": contractAddr.String(),
				"contract"	  : contract,
			}).Info("Smart Contract Deployement Transaction Sent Successful!")
		}
	}

	if err != nil {
		return nil, err
	}

	return tx.ID, err
}

func SetMinerKeyPair(key string) {
	minerPrivateKey = key
}

func GetMinerAddress() string {
	return minerPrivateKey
}

//add balance
func SendFromMiner(address core.Address, amount *common.Amount, bc *core.Blockchain) error {
	if !address.ValidateAddress() {
		return ErrInvalidAddress
	}

	if amount.Validate() != nil || amount.IsZero() {
		return ErrInvalidAmount
	}
	minerKeyPair := core.GetKeyPairByString(minerPrivateKey)
	minerWallet := &client.Wallet{}
	minerWallet.Key = minerKeyPair
	minerWallet.Addresses = append(minerWallet.Addresses, minerWallet.Key.GenerateAddress(false))

	pubKeyHash, _ := core.NewUserPubKeyHash(minerWallet.Key.PublicKey)
	utxos, err := core.LoadUTXOIndex(bc.GetDb()).GetUTXOsByAmount(pubKeyHash.GetPubKeyHash(), amount)
	if err != nil {
		return err
	}

	tx, err := core.NewUTXOTransaction(utxos, minerWallet.GetAddress(), address, amount, *minerWallet.GetKeyPair(), common.NewAmount(0),"")

	if err != nil {
		return err
	}

	bc.GetTxPool().Push(tx)

	return err

}

func GetWalletManager(path string) (*client.WalletManager, error) {
	fl := storage.NewFileLoader(path)
	wm := client.NewWalletManager(fl)
	err := wm.LoadFromFile()
	if err != nil {
		return nil, err
	}
	return wm, nil
}
