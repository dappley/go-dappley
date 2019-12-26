// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package transaction

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	logger "github.com/sirupsen/logrus"
	"strings"
	"time"
)

// Normal transaction
type TxNormal struct {
	*Transaction
}

// TxContract contains contract value
type TxContract struct {
	*Transaction
	Address account.Address
}

// Coinbase transaction, rewards to miner
type TxCoinbase struct {
	*Transaction
}

// GasReward transaction, rewards to miner during vm execution
type TxGasReward struct {
	*Transaction
}

// GasChange transaction, change value to from user
type TxGasChange struct {
	*Transaction
}

// Reward transaction, step reward
type TxReward struct {
	*Transaction
}

// Returns decorator of transaction
func NewTxDecorator(tx *Transaction) TxDecorator {
	// old data adapter
	adaptedTx := NewTxAdapter(tx)
	tx = adaptedTx.Transaction
	switch tx.Type {
	case TxTypeNormal:
		return &TxNormal{tx}
	case TxTypeContract:
		return NewTxContract(tx)
	case TxTypeCoinbase:
		return &TxCoinbase{tx}
	case TxTypeGasReward:
		return &TxGasReward{tx}
	case TxTypeGasChange:
		return &TxGasChange{tx}
	case TxTypeReward:
		return &TxReward{tx}
	}
	return nil
}

func (tx *TxNormal) Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error {
	return tx.sign(privKey, prevUtxos)
}

func (tx *TxNormal) IsNeedVerify() bool {
	return true
}

func (tx *TxNormal) Verify(prevUtxos []*utxo.UTXO, blockHeight uint64) error {
	return tx.verify(prevUtxos)
}

func (tx *TxContract) Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error {
	return tx.sign(privKey, prevUtxos)
}

func (tx *TxContract) IsNeedVerify() bool {
	return true
}

func (tx *TxContract) Verify(prevUtxos []*utxo.UTXO, blockHeight uint64) error {
	return nil
}

func (tx *TxCoinbase) Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error {
	return nil
}

func (tx *TxCoinbase) IsNeedVerify() bool {
	return true
}

func (tx *TxCoinbase) Verify(prevUtxos []*utxo.UTXO, blockHeight uint64) error {
	//TODO coinbase vout check need add tip
	if tx.Vout[0].Value.Cmp(Subsidy) < 0 {
		return errors.New("Transaction: subsidy check failed")
	}
	bh := binary.BigEndian.Uint64(tx.Vin[0].Signature)
	if blockHeight != bh {
		return fmt.Errorf("Transaction: block height check failed expected=%v actual=%v", blockHeight, bh)
	}
	return nil
}

func (tx *TxGasReward) Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error {
	return nil
}

func (tx *TxGasReward) IsNeedVerify() bool {
	return false
}

func (tx *TxGasReward) Verify(prevUtxos []*utxo.UTXO, blockHeight uint64) error {
	return nil
}

func (tx *TxGasChange) Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error {
	return nil
}

func (tx *TxGasChange) IsNeedVerify() bool {
	return false
}

func (tx *TxGasChange) Verify(prevUtxos []*utxo.UTXO, blockHeight uint64) error {
	return nil
}

func (tx *TxReward) Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error {
	return nil
}

func (tx *TxReward) IsNeedVerify() bool {
	return false
}

func (tx *TxReward) Verify(prevUtxos []*utxo.UTXO, blockHeight uint64) error {
	return nil
}

func NewTxContract(tx *Transaction) *TxContract {
	adaptedTx := NewTxAdapter(tx)
	if adaptedTx.isContract() {
		address := tx.Vout[ContractTxouputIndex].GetAddress()
		return &TxContract{tx, address}
	}
	return nil
}

// IsScheduleContract returns if the contract contains 'dapp_schedule'
func (ctx *TxContract) IsScheduleContract() bool {
	if !strings.Contains(ctx.GetContract(), scheduleFuncName) {
		return true
	}
	return false
}

//GetContract returns the smart contract code in a transaction
func (ctx *TxContract) GetContract() string {
	return ctx.Vout[ContractTxouputIndex].Contract
}

//GetContractPubKeyHash returns the smart contract pubkeyhash in a transaction
func (ctx *TxContract) GetContractPubKeyHash() account.PubKeyHash {
	return ctx.Vout[ContractTxouputIndex].PubKeyHash
}

// GasCountOfTxBase calculate the actual amount for a tx with data
func (ctx *TxContract) GasCountOfTxBase() (*common.Amount, error) {
	txGas := MinGasCountPerTransaction
	if dataLen := ctx.DataLen(); dataLen > 0 {
		dataGas := common.NewAmount(uint64(dataLen)).Mul(GasCountPerByte)
		baseGas := txGas.Add(dataGas)
		txGas = baseGas
	}
	return txGas, nil
}

// DataLen return the length of payload
func (ctx *TxContract) DataLen() int {
	return len([]byte(ctx.GetContract()))
}

// VerifyGas verifies if the transaction has the correct GasLimit and GasPrice
func (ctx *TxContract) VerifyGas(totalBalance *common.Amount) error {
	baseGas, err := ctx.GasCountOfTxBase()
	if err == nil {
		if ctx.GasLimit.Cmp(baseGas) < 0 {
			logger.WithFields(logger.Fields{
				"limit":       ctx.GasLimit,
				"acceptedGas": baseGas,
			}).Warn("Failed to check GasLimit >= txBaseGas.")
			// GasLimit is smaller than based tx gas, won't giveback the tx
			return ErrOutOfGasLimit
		}
	}

	limitedFee := ctx.GasLimit.Mul(ctx.GasPrice)
	if totalBalance.Cmp(limitedFee) < 0 {
		return ErrInsufficientBalance
	}
	return nil
}

// ToContractTx Returns structure of ContractTx
func ToContractTx(tx *Transaction) *TxContract {
	address := tx.Vout[ContractTxouputIndex].GetAddress()
	if tx.IsContract() {
		return &TxContract{tx, address}
	}
	if tx.Type != TxTypeDefault {
		return nil
	}
	txAdapter := NewTxAdapter(tx)
	if txAdapter.isContract() {
		return &TxContract{tx, address}
	}
	return nil
}

//GetContractAddress gets the smart contract's address if a transaction deploys a smart contract
func (tx *TxContract) GetContractAddress() account.Address {
	return tx.Address
}

//NewRewardTx creates a new transaction that gives reward to addresses according to the input rewards
func NewRewardTx(blockHeight uint64, rewards map[string]string) Transaction {

	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := transactionbase.TXInput{nil, -1, bh, RewardTxData}
	txOutputs := []transactionbase.TXOutput{}
	for address, amount := range rewards {
		amt, err := common.NewAmountFromString(amount)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"address": address,
				"amount":  amount,
			}).Warn("Transaction: failed to parse reward amount")
		}
		acc := account.NewContractAccountByAddress(account.NewAddress(address))
		txOutputs = append(txOutputs, *transactionbase.NewTXOutput(amt, acc))
	}
	tx := Transaction{nil, []transactionbase.TXInput{txin}, txOutputs, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), time.Now().UnixNano() / 1e6, TxTypeReward}

	tx.ID = tx.Hash()

	return tx
}

// NewGasRewardTx returns a reward to miner, earned for contract execution gas fee
func NewGasRewardTx(to *account.TransactionAccount, blockHeight uint64, actualGasCount *common.Amount, gasPrice *common.Amount, uniqueNum int) (Transaction, error) {
	fee := actualGasCount.Mul(gasPrice)
	txin := transactionbase.TXInput{nil, -1, getUniqueByte(blockHeight, uniqueNum), gasRewardData}
	txout := transactionbase.NewTXOutput(fee, to)
	tx := Transaction{nil, []transactionbase.TXInput{txin}, []transactionbase.TXOutput{*txout}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), time.Now().UnixNano() / 1e6, TxTypeGasReward}
	tx.ID = tx.Hash()
	return tx, nil
}

// NewGasChangeTx returns a change to contract invoker, pay for the change of unused gas
func NewGasChangeTx(to *account.TransactionAccount, blockHeight uint64, actualGasCount *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, uniqueNum int) (Transaction, error) {
	if gasLimit.Cmp(actualGasCount) <= 0 {
		return Transaction{}, ErrNoGasChange
	}
	change, err := gasLimit.Sub(actualGasCount)

	if err != nil {
		return Transaction{}, err
	}
	changeValue := change.Mul(gasPrice)

	txin := transactionbase.TXInput{nil, -1, getUniqueByte(blockHeight, uniqueNum), gasChangeData}
	txout := transactionbase.NewTXOutput(changeValue, to)
	tx := Transaction{nil, []transactionbase.TXInput{txin}, []transactionbase.TXOutput{*txout}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), time.Now().UnixNano() / 1e6, TxTypeGasChange}

	tx.ID = tx.Hash()
	return tx, nil
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to account.Address, data string, blockHeight uint64, tip *common.Amount) Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}
	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))
	toAccount := account.NewContractAccountByAddress(to)
	txin := transactionbase.TXInput{nil, -1, bh, []byte(data)}
	txout := transactionbase.NewTXOutput(Subsidy.Add(tip), toAccount)
	tx := Transaction{nil, []transactionbase.TXInput{txin}, []transactionbase.TXOutput{*txout}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), time.Now().UnixNano() / 1e6, TxTypeCoinbase}
	tx.ID = tx.Hash()

	return tx
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(utxos []*utxo.UTXO, sendTxParam SendTxParam) (Transaction, error) {
	fromAccount := account.NewContractAccountByAddress(sendTxParam.From)
	toAccount := account.NewContractAccountByAddress(sendTxParam.To)
	sum := calculateUtxoSum(utxos)
	change, err := calculateChange(sum, sendTxParam.Amount, sendTxParam.Tip, sendTxParam.GasLimit, sendTxParam.GasPrice)
	if err != nil {
		return Transaction{}, err
	}
	txType := TxTypeNormal
	if sendTxParam.Contract != "" {
		txType = TxTypeContract
	}
	tx := Transaction{
		nil,
		prepareInputLists(utxos, sendTxParam.SenderKeyPair.GetPublicKey(), nil),
		prepareOutputLists(fromAccount, toAccount, sendTxParam.Amount, change, sendTxParam.Contract),
		sendTxParam.Tip,
		sendTxParam.GasLimit,
		sendTxParam.GasPrice,
		time.Now().UnixNano() / 1e6,
		txType,
	}
	tx.ID = tx.Hash()

	err = tx.sign(sendTxParam.SenderKeyPair.GetPrivateKey(), utxos)
	if err != nil {
		return Transaction{}, err
	}

	return tx, nil
}

func NewSmartContractDestoryTX(utxos []*utxo.UTXO, contractAddr account.Address, sourceTXID []byte) Transaction {
	sum := calculateUtxoSum(utxos)
	tips := common.NewAmount(0)
	gasLimit := common.NewAmount(0)
	gasPrice := common.NewAmount(0)

	tx, _ := NewContractTransferTX(utxos, contractAddr, account.NewAddress(SCDestroyAddress), sum, tips, gasLimit, gasPrice, sourceTXID)
	return tx
}

func NewContractTransferTX(utxos []*utxo.UTXO, contractAddr, toAddr account.Address, amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, sourceTXID []byte) (Transaction, error) {
	contractAccount := account.NewContractAccountByAddress(contractAddr)
	toAccount := account.NewContractAccountByAddress(toAddr)
	if !contractAccount.IsValid() {
		return Transaction{}, account.ErrInvalidAddress
	}
	if isContract, err := contractAccount.GetPubKeyHash().IsContract(); !isContract {
		return Transaction{}, err
	}

	sum := calculateUtxoSum(utxos)
	change, err := calculateChange(sum, amount, tip, gasLimit, gasPrice)
	if err != nil {
		return Transaction{}, err
	}

	// Intentionally set PubKeyHash as PubKey (to recognize it is from contract) and sourceTXID as signature in Vin
	tx := Transaction{
		nil,
		prepareInputLists(utxos, contractAccount.GetPubKeyHash(), sourceTXID),
		prepareOutputLists(contractAccount, toAccount, amount, change, ""),
		tip,
		gasLimit,
		gasPrice,
		time.Now().UnixNano() / 1e6,
		TxTypeNormal,
	}
	tx.ID = tx.Hash()

	return tx, nil
}
