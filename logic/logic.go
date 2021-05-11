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

	"github.com/dappley/go-dappley/logic/ltransaction"

	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/logic/transactionpool"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"

	"github.com/dappley/go-dappley/wallet"
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
func CreateBlockchain(address account.Address, db storage.Storage, libPolicy lblockchain.LIBPolicy, txPool *transactionpool.TransactionPool, blkSizeLimit int) (*lblockchain.Blockchain, error) {
	addressAccount := account.NewTransactionAccountByAddress(address)
	if !addressAccount.IsValid() {
		return nil, ErrInvalidAddress
	}

	bc := lblockchain.CreateBlockchain(address, db, libPolicy, txPool, blkSizeLimit)

	return bc, nil
}

// Returns default account file path or first argument of argument vector
func getAccountFilePath(argv []string) string {
	if len(argv) == 1 {
		return argv[0]
	}
	return wallet.GetAccountFilePath()
}

//Get lock flag
func IsAccountLocked(optionalAccountFilePath ...string) (bool, error) {
	am, err := GetAccountManager(getAccountFilePath(optionalAccountFilePath))
	return am.Locked, err
}

//Tell if the file empty or not exist
func IsAccountEmpty(optionalAccountFilePath ...string) (bool, error) {
	accountFilePath := getAccountFilePath(optionalAccountFilePath)

	am, _ := GetAccountManager(accountFilePath)
	if am == nil {
		return true, nil
	}
	if am.IsEmpty() {
		return true, nil
	}
	return false, nil
}

//Set lock flag
func SetLockAccount(optionalAccountFilePath ...string) error {
	am, err1 := GetAccountManager(getAccountFilePath(optionalAccountFilePath))

	if err1 != nil {
		return err1
	}

	empty := am.IsEmpty()
	if empty {
		return nil
	}

	am.Locked = true
	am.SaveAccountToFile()
	return nil
}

//Set unlock and timer
func SetUnLockAccount(optionalAccountFilePath ...string) error {
	am, err := GetAccountManager(getAccountFilePath(optionalAccountFilePath))
	if err != nil {
		return err
	}
	am.SetUnlockTimer(unlockduration)
	return nil
}

//create a account with passphrase
func CreateAccountWithPassphrase(password string, optionalAccountFilePath ...string) (*account.Account, error) {
	am, err := GetAccountManager(getAccountFilePath(optionalAccountFilePath))
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	if len(am.Accounts) > 0 && am.PassPhrase != nil {
		err = bcrypt.CompareHashAndPassword(am.PassPhrase, []byte(password))
		if err != nil {
			return nil, ErrPasswordNotMatch
		}
		account := account.NewAccount()
		am.AddAccount(account)
		am.SaveAccountToFile()
		return account, err
	}
	passBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	am.PassPhrase = passBytes
	logger.Info("Account password is set!")
	account := account.NewAccount()
	am.AddAccount(account)
	am.Locked = true
	am.SaveAccountToFile()
	return account, err
}

//create a account
func CreateAccount() (*account.Account, error) {
	am, err := GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		return nil, err
	}

	account := account.NewAccount()
	if len(am.Accounts) == 0 {
		am.Locked = true
	}
	am.AddAccount(account)
	am.SaveAccountToFile()
	return account, err
}

//Get duration
func GetUnlockDuration() time.Duration {
	return unlockduration
}

//get balance
func GetBalance(address account.Address, bc *lblockchain.Blockchain) (*common.Amount, error) {
	acc := account.NewTransactionAccountByAddress(address)
	if acc.IsValid() == false {
		return common.NewAmount(0), ErrInvalidAddress
	}

	balance := common.NewAmount(0)
	utxoIndex := lutxo.NewUTXOIndex(bc.GetUtxoCache())
	utxos := utxoIndex.GetAllUTXOsByPubKeyHash(acc.GetPubKeyHash())
	for _, utxo := range utxos.Indices {
		balance = balance.Add(utxo.Value)
	}

	return balance, nil
}

func Send(senderAccount *account.Account, to account.Address, amount *common.Amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, contract string, bc *lblockchain.Blockchain) ([]byte, string, error) {
	sendTxParam := transaction.NewSendTxParam(senderAccount.GetAddress(), senderAccount.GetKeyPair(), to, amount, tip, gasLimit, gasPrice, contract)
	return sendTo(sendTxParam, bc)
}

func SetMinerKeyPair(key string) {
	minerPrivateKey = key
}

func GetMinerAddress() string {
	return minerPrivateKey
}

//add balance
func SendFromMiner(address account.Address, amount *common.Amount, bc *lblockchain.Blockchain) ([]byte, string, error) {
	minerAccount := account.NewAccountByPrivateKey(minerPrivateKey)
	sendTxParam := transaction.NewSendTxParam(minerAccount.GetAddress(), minerAccount.GetKeyPair(), address, amount, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	return sendTo(sendTxParam, bc)
}

func GetAccountManager(path string) (*wallet.AccountManager, error) {
	fl := storage.NewFileLoader(path)
	am := wallet.NewAccountManager(fl)
	err := am.LoadFromFile()
	if err != nil {
		return nil, err
	}
	return am, nil
}

func sendTo(sendTxParam transaction.SendTxParam, bc *lblockchain.Blockchain) ([]byte, string, error) {
	fromAccount := account.NewTransactionAccountByAddress(sendTxParam.From)
	toAccount := account.NewTransactionAccountByAddress(sendTxParam.To)
	if !fromAccount.IsValid() {
		return nil, "", ErrInvalidSenderAddress
	}

	//Contract deployment transaction does not need to validate to address
	if !toAccount.IsValid() && sendTxParam.Contract == "" {
		return nil, "", ErrInvalidRcverAddress
	}

	if sendTxParam.Amount.Validate() != nil || sendTxParam.Amount.IsZero() {
		return nil, "", ErrInvalidAmount
	}

	acc := account.NewAccountByKey(sendTxParam.SenderKeyPair)
	utxoIndex := lutxo.NewUTXOIndex(bc.GetUtxoCache())
	if !utxoIndex.UpdateUtxos(bc.GetTxPool().GetAllTransactions()) {
		logger.Warn("sendTo error")
	}
	utxos, err := utxoIndex.GetUTXOsAccordingToAmount([]byte(acc.GetPubKeyHash()), sendTxParam.TotalCost())
	if err != nil {
		return nil, "", err
	}

	tx, err := ltransaction.NewUTXOTransaction(utxos, sendTxParam)

	bc.GetTxPool().Push(tx)
	bc.GetTxPool().BroadcastTx(&tx)

	contractAddr := account.NewAddress("")
	if tx.Type == transaction.TxTypeContract {
		contractAddr = ltransaction.NewTxContract(&tx).GetContractAddress()
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
