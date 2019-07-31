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

	"github.com/dappley/go-dappley/core/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic/account_logic"
	"github.com/dappley/go-dappley/network"
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
func CreateBlockchain(address client.Address, db storage.Storage, consensus core.Consensus, transactionPoolLimit uint32, scManager *vm.V8EngineManager, blkSizeLimit int) (*core.Blockchain, error) {
	if !address.IsValid() {
		return nil, ErrInvalidAddress
	}

	bc := core.CreateBlockchain(address, db, consensus, transactionPoolLimit, scManager, blkSizeLimit)

	return bc, nil
}

//create a account from path
func CreateAccount(path string, password string) (*client.Account, error) {
	if len(path) == 0 {
		return nil, ErrPathEmpty
	}

	if len(password) == 0 {
		return nil, ErrPasswordEmpty
	}

	fl := storage.NewFileLoader(path)
	am := account_logic.NewAccountManager(fl)
	passBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	am.PassPhrase = passBytes
	am.Locked = true
	err = am.LoadFromFile()
	account := client.NewAccount()
	am.AddAccount(account)
	am.SaveAccountToFile()

	return account, err
}

//get account
func GetAccount() (*client.Account, error) {
	am, err := GetAccountManager(account_logic.GetAccountFilePath())
	empty, err := am.IsFileEmpty()
	if empty {
		return nil, nil
	}
	if len(am.Accounts) > 0 {
		return am.Accounts[0], err
	}
	return nil, err
}

// Returns default account file path or first argument of argument vector
func getAccountFilePath(argv []string) string {
	if len(argv) == 1 {
		return argv[0]
	}
	return account_logic.GetAccountFilePath()
}

//Get lock flag
func IsAccountLocked(optionalAccountFilePath ...string) (bool, error) {
	am, err := GetAccountManager(getAccountFilePath(optionalAccountFilePath))
	return am.Locked, err
}

//Tell if the file empty or not exist
func IsAccountEmpty(optionalAccountFilePath ...string) (bool, error) {
	accountFilePath := getAccountFilePath(optionalAccountFilePath)

	if account_logic.Exists(accountFilePath) {
		am, _ := GetAccountManager(accountFilePath)
		if len(am.Accounts) == 0 {
			return true, nil
		}
		return am.IsFileEmpty()
	}
	return true, nil
}

//Set lock flag
func SetLockAccount(optionalAccountFilePath ...string) error {
	am, err1 := GetAccountManager(getAccountFilePath(optionalAccountFilePath))

	if err1 != nil {
		return err1
	}

	empty, err2 := am.IsFileEmpty()

	if err2 != nil {
		return err2
	}

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
func CreateAccountWithpassphrase(password string, optionalAccountFilePath ...string) (*client.Account, error) {
	am, err := GetAccountManager(getAccountFilePath(optionalAccountFilePath))
	if err != nil {
		return nil, err
	}

	if len(am.Accounts) > 0 && am.PassPhrase != nil {
		err = bcrypt.CompareHashAndPassword(am.PassPhrase, []byte(password))
		if err != nil {
			return nil, ErrPasswordNotMatch
		}
		account := client.NewAccount()
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
	account := client.NewAccount()
	am.AddAccount(account)
	am.Locked = true
	am.SaveAccountToFile()
	return account, err

}

//create a account
func AddAccount() (*client.Account, error) {
	am, err := GetAccountManager(account_logic.GetAccountFilePath())
	if err != nil {
		return nil, err
	}

	account := client.NewAccount()
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
func GetBalance(address client.Address, bc *core.Blockchain) (*common.Amount, error) {
	pubKeyHash, valid := client.GeneratePubKeyHashByAddress(address)
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

func Send(senderAccount *client.Account, to client.Address, amount *common.Amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, contract string, bc *core.Blockchain, node *network.Node) ([]byte, string, error) {
	sendTxParam := core.NewSendTxParam(senderAccount.GetKeyPair().GenerateAddress(), senderAccount.GetKeyPair(), to, amount, tip, gasLimit, gasPrice, contract)
	return sendTo(sendTxParam, bc, node)
}

func SetMinerKeyPair(key string) {
	minerPrivateKey = key
}

func GetMinerAddress() string {
	return minerPrivateKey
}

//add balance
func SendFromMiner(address client.Address, amount *common.Amount, bc *core.Blockchain, node *network.Node) ([]byte, string, error) {
	minerKeyPair := client.GetKeyPairByString(minerPrivateKey)
	sendTxParam := core.NewSendTxParam(minerKeyPair.GenerateAddress(), minerKeyPair, address, amount, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	return sendTo(sendTxParam, bc, node)
}

func GetAccountManager(path string) (*account_logic.AccountManager, error) {
	fl := storage.NewFileLoader(path)
	am := account_logic.NewAccountManager(fl)
	err := am.LoadFromFile()
	if err != nil {
		return nil, err
	}
	return am, nil
}

func sendTo(sendTxParam core.SendTxParam, bc *core.Blockchain, node *network.Node) ([]byte, string, error) {
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

	pubKeyHash, _ := client.NewUserPubKeyHash(sendTxParam.SenderKeyPair.PublicKey)
	utxoIndex := core.NewUTXOIndex(bc.GetUtxoCache())

	utxoIndex.UpdateUtxoState(bc.GetTxPool().GetAllTransactions())

	utxos, err := utxoIndex.GetUTXOsByAmount([]byte(pubKeyHash), sendTxParam.TotalCost())
	if err != nil {
		return nil, "", err
	}

	tx, err := core.NewUTXOTransaction(utxos, sendTxParam)

	bc.GetTxPool().Push(tx)
	if node != nil {
		node.TxBroadcast(&tx)
	}
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
