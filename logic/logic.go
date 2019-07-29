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
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/vm"
	logger "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const unlockduration = 300 * time.Second

var minerPrivateKey string
var (
	ErrInvalidAmount        = errors.New("invalid amount (must be > 0)")
	ErrInvalidAddress       = errors.New("invalid address")
	ErrInvalidSenderAddress = errors.New("invalid sender address")
	ErrInvalidRcverAddress  = errors.New("invalid receiver address")
	ErrPasswordNotMatch     = errors.New("password is incorrect")
	ErrPathEmpty            = errors.New("empty path")
	ErrPasswordEmpty        = errors.New("empty password")
)

//create a blockchain
func CreateBlockchain(address core.Address, db storage.Storage, consensus core.Consensus, txPool *core.TransactionPool, scManager *vm.V8EngineManager, blkSizeLimit int) (*core.Blockchain, error) {
	if !address.IsValid() {
		return nil, ErrInvalidAddress
	}

	bc := core.CreateBlockchain(address, db, consensus, txPool, scManager, blkSizeLimit)

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
	}
	return nil, err
}

// Returns default wallet file path or first argument of argument vector
func getWalletFilePath(argv []string) string {
	if len(argv) == 1 {
		return argv[0]
	}
	return client.GetWalletFilePath()
}

//Get lock flag
func IsWalletLocked(optionalWalletFilePath ...string) (bool, error) {
	wm, err := GetWalletManager(getWalletFilePath(optionalWalletFilePath))
	return wm.Locked, err
}

//Tell if the file empty or not exist
func IsWalletEmpty(optionalWalletFilePath ...string) (bool, error) {
	walletFilePath := getWalletFilePath(optionalWalletFilePath)

	if client.Exists(walletFilePath) {
		wm, _ := GetWalletManager(walletFilePath)
		if len(wm.Wallets) == 0 {
			return true, nil
		}
		return wm.IsFileEmpty()
	}
	return true, nil
}

//Set lock flag
func SetLockWallet(optionalWalletFilePath ...string) error {
	wm, err1 := GetWalletManager(getWalletFilePath(optionalWalletFilePath))

	if err1 != nil {
		return err1
	}

	empty, err2 := wm.IsFileEmpty()

	if err2 != nil {
		return err2
	}

	if empty {
		return nil
	}

	wm.Locked = true
	wm.SaveWalletToFile()
	return nil
}

//Set unlock and timer
func SetUnLockWallet(optionalWalletFilePath ...string) error {
	wm, err := GetWalletManager(getWalletFilePath(optionalWalletFilePath))
	if err != nil {
		return err
	}
	wm.SetUnlockTimer(unlockduration)
	return nil
}

//create a wallet with passphrase
func CreateWalletWithpassphrase(password string, optionalWalletFilePath ...string) (*client.Wallet, error) {
	wm, err := GetWalletManager(getWalletFilePath(optionalWalletFilePath))
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

	}
	passBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	wm.PassPhrase = passBytes
	logger.Info("Wallet password is set!")
	wallet := client.NewWallet()
	wm.AddWallet(wallet)
	wm.Locked = true
	wm.SaveWalletToFile()
	return wallet, err

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
func GetBalance(address core.Address, bc *core.Blockchain) (*common.Amount, error) {
	pubKeyHash, valid := address.GetPubKeyHash()
	if valid == false {
		return common.NewAmount(0), ErrInvalidAddress
	}

	balance := common.NewAmount(0)
	utxoIndex := core.NewUTXOIndex(bc.GetUtxoCache())
	utxos := utxoIndex.GetAllUTXOsByPubKeyHash(pubKeyHash)
	for _, utxo := range utxos.Indices {
		balance = balance.Add(utxo.Value)
	}

	return balance, nil
}

func Send(senderWallet *client.Wallet, to core.Address, amount *common.Amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, contract string, bc *core.Blockchain) ([]byte, string, error) {
	sendTxParam := core.NewSendTxParam(senderWallet.GetAddress(), senderWallet.GetKeyPair(), to, amount, tip, gasLimit, gasPrice, contract)
	return sendTo(sendTxParam, bc)
}

func SetMinerKeyPair(key string) {
	minerPrivateKey = key
}

func GetMinerAddress() string {
	return minerPrivateKey
}

//add balance
func SendFromMiner(address core.Address, amount *common.Amount, bc *core.Blockchain) ([]byte, string, error) {
	minerKeyPair := core.GetKeyPairByString(minerPrivateKey)
	sendTxParam := core.NewSendTxParam(minerKeyPair.GenerateAddress(false), minerKeyPair, address, amount, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	return sendTo(sendTxParam, bc)
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

func sendTo(sendTxParam core.SendTxParam, bc *core.Blockchain) ([]byte, string, error) {
	if !sendTxParam.From.IsValid() {
		return nil, "", ErrInvalidSenderAddress
	}

	//Contract deployment transaction does not need to validate to address
	if !sendTxParam.To.IsValid() && sendTxParam.Contract == "" {
		return nil, "", ErrInvalidRcverAddress
	}

	if sendTxParam.Amount.Validate() != nil || sendTxParam.Amount.IsZero() {
		return nil, "", ErrInvalidAmount
	}

	pubKeyHash, _ := core.NewUserPubKeyHash(sendTxParam.SenderKeyPair.PublicKey)
	utxoIndex := core.NewUTXOIndex(bc.GetUtxoCache())

	utxoIndex.UpdateUtxoState(bc.GetTxPool().GetAllTransactions())

	utxos, err := utxoIndex.GetUTXOsByAmount([]byte(pubKeyHash), sendTxParam.TotalCost())
	if err != nil {
		return nil, "", err
	}

	tx, err := core.NewUTXOTransaction(utxos, sendTxParam)

	bc.GetTxPool().Push(tx)
	bc.GetTxPool().BroadcastTx(&tx)

	contractAddr := tx.GetContractAddress()
	if contractAddr.String() != "" {
		if sendTxParam.To.String() == contractAddr.String() {
			logger.WithFields(logger.Fields{
				"contract_address": contractAddr.String(),
				"data":             sendTxParam.Contract,
			}).Info("Smart contract invocation transaction is sent.")
		} else {
			logger.WithFields(logger.Fields{
				"contract_address": contractAddr.String(),
				"contract":         sendTxParam.Contract,
			}).Info("Smart contract deployment transaction is sent.")
		}
	}

	if err != nil {
		return nil, "", err
	}

	return tx.ID, contractAddr.String(), err
}
