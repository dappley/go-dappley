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
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/transaction_base"
	transactionbasepb "github.com/dappley/go-dappley/core/transaction_base/pb"
	"github.com/dappley/go-dappley/crypto/byteutils"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

var Subsidy = common.NewAmount(10000000)

const (
	ContractTxouputIndex = 0
	scheduleFuncName     = "dapp_schedule"
)

var RewardTxData = []byte("Distribute X Rewards")
var gasRewardData = []byte("Miner Gas Rewards")
var gasChangeData = []byte("Unspent Gas Change")

var (
	// MinGasCountPerTransaction default gas for normal transaction
	MinGasCountPerTransaction = common.NewAmount(20000)
	// GasCountPerByte per byte of data attached to a transaction gas cost
	GasCountPerByte        = common.NewAmount(1)
	ErrOutOfGasLimit       = errors.New("out of gas limit")
	ErrInsufficientFund    = errors.New("transaction: insufficient balance")
	ErrInvalidAmount       = errors.New("transaction: invalid amount (must be > 0)")
	ErrTXInputNotFound     = errors.New("transaction: transaction input not found")
	ErrNewUserPubKeyHash   = errors.New("transaction: create pubkeyhash error")
	ErrNoGasChange         = errors.New("transaction: all of Gas have been consumed")
	ErrInsufficientBalance = errors.New("Transaction: insufficient balance, cannot pay for GasLimit")
)

type Transaction struct {
	ID       []byte
	Vin      []transaction_base.TXInput
	Vout     []transaction_base.TXOutput
	Tip      *common.Amount
	GasLimit *common.Amount
	GasPrice *common.Amount
}

// ContractTx contains contract value
type ContractTx struct {
	Transaction
}

type TxIndex struct {
	BlockId    []byte
	BlockIndex int
}

//SendTxParam Transaction parameters
type SendTxParam struct {
	From          account.Address
	SenderKeyPair *account.KeyPair
	To            account.Address
	Amount        *common.Amount
	Tip           *common.Amount
	GasLimit      *common.Amount
	GasPrice      *common.Amount
	Contract      string
}

// NewSendTxParam Returns SendTxParam object
func NewSendTxParam(from account.Address, senderKeyPair *account.KeyPair, to account.Address, amount *common.Amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, contract string) SendTxParam {
	return SendTxParam{from, senderKeyPair, to, amount, tip, gasLimit, gasPrice, contract}
}

// TotalCost returns total cost of utxo value in this transaction
func (st SendTxParam) TotalCost() *common.Amount {
	var totalAmount = st.Amount
	if st.Tip != nil {
		totalAmount = totalAmount.Add(st.Tip)
	}
	if st.GasLimit != nil {
		limitedFee := st.GasLimit.Mul(st.GasPrice)
		totalAmount = totalAmount.Add(limitedFee)
	}
	return totalAmount
}

//ToContractTx Returns structure of ContractTx
func (tx *Transaction) ToContractTx() *ContractTx {
	if !tx.IsContract() {
		return nil
	}
	return &ContractTx{*tx}
}

func (tx *Transaction) IsCoinbase() bool {

	if !tx.isVinCoinbase() {
		return false
	}

	if len(tx.Vout) != 1 {
		return false
	}

	if len(tx.Vin[0].PubKey) == 0 {
		return false
	}

	if bytes.Equal(tx.Vin[0].PubKey, RewardTxData) {
		return false
	}

	if bytes.Equal(tx.Vin[0].PubKey, gasRewardData) {
		return false
	}

	if bytes.Equal(tx.Vin[0].PubKey, gasChangeData) {
		return false
	}

	return true
}

// IsRewardTx returns if the transaction is about the step reward
func (tx *Transaction) IsRewardTx() bool {

	if !tx.isVinCoinbase() {
		return false
	}

	if !bytes.Equal(tx.Vin[0].PubKey, RewardTxData) {
		return false
	}

	return true
}

// IsRewardTx returns if the transaction is about the gas reward to miner after smart contract execution
func (tx *Transaction) IsGasRewardTx() bool {

	if !tx.isVinCoinbase() {
		return false
	}

	if len(tx.Vout) != 1 {
		return false
	}

	if !bytes.Equal(tx.Vin[0].PubKey, gasRewardData) {
		return false
	}
	return true
}

// IsRewardTx returns if the transaction is about the gas change to from address after smart contract execution
func (tx *Transaction) IsGasChangeTx() bool {

	if !tx.isVinCoinbase() {
		return false
	}

	if len(tx.Vout) != 1 {
		return false
	}

	if !bytes.Equal(tx.Vin[0].PubKey, gasChangeData) {
		return false
	}

	return true
}

// IsContract returns true if tx deploys/executes a smart contract; false otherwise
func (tx *Transaction) IsContract() bool {
	if len(tx.Vout) == 0 {
		return false
	}
	isContract, _ := tx.Vout[ContractTxouputIndex].PubKeyHash.IsContract()
	return isContract
}

func (ctx *ContractTx) IsContract() bool {
	return true
}

func (ctx *ContractTx) IsExecutionContract() bool {
	if !strings.Contains(ctx.GetContract(), scheduleFuncName) {
		return true
	}
	return false
}

func (tx *Transaction) isVinCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

//GetToHashBytes Get bytes for hash
func (tx *Transaction) GetToHashBytes() []byte {
	var tempBytes []byte

	for _, vin := range tx.Vin {
		tempBytes = bytes.Join([][]byte{
			tempBytes,
			vin.Txid,
			byteutils.FromInt32(int32(vin.Vout)),
			vin.PubKey,
			vin.Signature,
		}, []byte{})
	}
	for _, vout := range tx.Vout {
		tempBytes = bytes.Join([][]byte{
			tempBytes,
			vout.Value.Bytes(),
			[]byte(vout.PubKeyHash),
			[]byte(vout.Contract),
		}, []byte{})
	}

	if tx.Tip != nil {
		tempBytes = append(tempBytes, tx.Tip.Bytes()...)
	}
	if tx.GasLimit != nil {
		tempBytes = append(tempBytes, tx.GasLimit.Bytes()...)
	}
	if tx.GasPrice != nil {
		tempBytes = append(tempBytes, tx.GasPrice.Bytes()...)
	}

	return tempBytes
}

// Hash returns the hash of the Transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	hash = sha256.Sum256(tx.GetToHashBytes())

	return hash[:]
}

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing
func (tx *Transaction) TrimmedCopy(withSignature bool) Transaction {
	var inputs []transaction_base.TXInput
	var outputs []transaction_base.TXOutput
	var pubkey []byte

	for _, vin := range tx.Vin {
		if withSignature {
			pubkey = vin.PubKey
		} else {
			pubkey = nil
		}
		inputs = append(inputs, transaction_base.TXInput{vin.Txid, vin.Vout, nil, pubkey})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, transaction_base.TXOutput{vout.Value, vout.PubKeyHash, vout.Contract})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip, tx.GasLimit, tx.GasPrice}

	return txCopy
}

func (tx *Transaction) DeepCopy() Transaction {
	var inputs []transaction_base.TXInput
	var outputs []transaction_base.TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, transaction_base.TXInput{vin.Txid, vin.Vout, vin.Signature, vin.PubKey})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, transaction_base.TXOutput{vout.Value, vout.PubKeyHash, vout.Contract})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip, tx.GasLimit, tx.GasPrice}

	return txCopy
}

// VerifyID verifies if the transaction ID is the hash of the transaction
func (tx *Transaction) VerifyID() (bool, error) {
	txCopy := tx.TrimmedCopy(true)
	if bytes.Equal(tx.ID, (&txCopy).Hash()) {
		return true, nil
	} else {
		return false, errors.New("Transaction: ID is invalid")
	}
}

// VerifyAmount verifies if the transaction has the correct vout value
func (tx *Transaction) VerifyAmount(totalPrev *common.Amount, totalVoutValue *common.Amount) (bool, error) {
	//TotalVin amount must equal or greater than total vout
	if totalPrev.Cmp(totalVoutValue) >= 0 {
		return true, nil
	}
	return false, errors.New("Transaction: amount is invalid")
}

//VerifyTip verifies if the transaction has the correct tip
func (tx *Transaction) VerifyTip(totalPrev *common.Amount, totalVoutValue *common.Amount) (bool, error) {
	sub, err := totalPrev.Sub(totalVoutValue)
	if err != nil {
		return false, err
	}
	if tx.Tip.Cmp(sub) > 0 {
		return false, errors.New("Transaction: tip is invalid")
	}
	return true, nil
}

// VerifyGas verifies if the transaction has the correct GasLimit and GasPrice
func (ctx *ContractTx) VerifyGas(totalBalance *common.Amount) (bool, error) {
	baseGas, err := ctx.GasCountOfTxBase()
	if err == nil {
		if ctx.GasLimit.Cmp(baseGas) < 0 {
			logger.WithFields(logger.Fields{
				"limit":       ctx.GasLimit,
				"acceptedGas": baseGas,
			}).Warn("Failed to check GasLimit >= txBaseGas.")
			// GasLimit is smaller than based tx gas, won't giveback the tx
			return false, ErrOutOfGasLimit
		}
	}

	limitedFee := ctx.GasLimit.Mul(ctx.GasPrice)
	if totalBalance.Cmp(limitedFee) < 0 {
		return false, ErrInsufficientBalance
	}
	return true, nil
}

//CalculateTotalVoutValue returns total amout of transaction's vout
func (tx *Transaction) CalculateTotalVoutValue() (*common.Amount, bool) {
	totalVout := &common.Amount{}
	for _, vout := range tx.Vout {
		if vout.Value == nil || vout.Value.Validate() != nil {
			return nil, false
		}
		totalVout = totalVout.Add(vout.Value)
	}
	return totalVout, true
}

//NewRewardTx creates a new transaction that gives reward to addresses according to the input rewards
func NewRewardTx(blockHeight uint64, rewards map[string]string) Transaction {

	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := transaction_base.TXInput{nil, -1, bh, RewardTxData}
	txOutputs := []transaction_base.TXOutput{}
	for address, amount := range rewards {
		amt, err := common.NewAmountFromString(amount)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"address": address,
				"amount":  amount,
			}).Warn("Transaction: failed to parse reward amount")
		}
		txOutputs = append(txOutputs, *transaction_base.NewTXOutput(amt, account.NewAddress(address)))
	}
	tx := Transaction{nil, []transaction_base.TXInput{txin}, txOutputs, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0)}
	tx.ID = tx.Hash()

	return tx
}

func NewTransactionByVin(vinTxId []byte, vinVout int, vinPubkey []byte, voutValue uint64, voutPubKeyHash account.PubKeyHash, tip uint64) Transaction {
	tx := Transaction{
		ID: nil,
		Vin: []transaction_base.TXInput{
			{vinTxId, vinVout, nil, vinPubkey},
		},
		Vout: []transaction_base.TXOutput{
			{common.NewAmount(voutValue), voutPubKeyHash, ""},
		},
		Tip: common.NewAmount(tip),
	}
	tx.ID = tx.Hash()
	return tx
}

// NewGasRewardTx returns a reward to miner, earned for contract execution gas fee
func NewGasRewardTx(to account.Address, blockHeight uint64, actualGasCount *common.Amount, gasPrice *common.Amount) (Transaction, error) {
	fee := actualGasCount.Mul(gasPrice)
	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := transaction_base.TXInput{nil, -1, bh, gasRewardData}
	txout := transaction_base.NewTXOutput(fee, to)
	tx := Transaction{nil, []transaction_base.TXInput{txin}, []transaction_base.TXOutput{*txout}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0)}
	tx.ID = tx.Hash()
	return tx, nil
}

// NewGasChangeTx returns a change to contract invoker, pay for the change of unused gas
func NewGasChangeTx(to account.Address, blockHeight uint64, actualGasCount *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount) (Transaction, error) {
	if gasLimit.Cmp(actualGasCount) <= 0 {
		return Transaction{}, ErrNoGasChange
	}
	change, err := gasLimit.Sub(actualGasCount)

	if err != nil {
		return Transaction{}, err
	}
	changeValue := change.Mul(gasPrice)
	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := transaction_base.TXInput{nil, -1, bh, gasChangeData}
	txout := transaction_base.NewTXOutput(changeValue, to)
	tx := Transaction{nil, []transaction_base.TXInput{txin}, []transaction_base.TXOutput{*txout}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0)}
	tx.ID = tx.Hash()
	return tx, nil
}

//GetContractAddress gets the smart contract's address if a transaction deploys a smart contract
func (tx *Transaction) GetContractAddress() account.Address {
	ctx := tx.ToContractTx()
	if ctx == nil {
		return account.NewAddress("")
	}

	return ctx.GetContractPubKeyHash().GenerateAddress()
}

//GetContract returns the smart contract code in a transaction
func (ctx *ContractTx) GetContract() string {
	return ctx.Vout[ContractTxouputIndex].Contract
}

//GetContractPubKeyHash returns the smart contract pubkeyhash in a transaction
func (ctx *ContractTx) GetContractPubKeyHash() account.PubKeyHash {
	return ctx.Vout[ContractTxouputIndex].PubKeyHash
}

func (tx *Transaction) MatchRewards(rewardStorage map[string]string) bool {

	if tx == nil {
		logger.Debug("Transaction: does not exist")
		return false
	}

	if !tx.IsRewardTx() {
		logger.Debug("Transaction: is not a reward transaction")
		return false
	}

	for _, vout := range tx.Vout {
		if !vout.IsFoundInRewardStorage(rewardStorage) {
			return false
		}
	}
	return len(rewardStorage) == len(tx.Vout)
}

// String returns a human-readable representation of a transaction
func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("\n--- Transaction %x:", tx.ID))

	for i, input := range tx.Vin {

		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", []byte(output.PubKeyHash)))
		lines = append(lines, fmt.Sprintf("       Contract: %s", output.Contract))
	}
	lines = append(lines, "\n")

	return strings.Join(lines, "\n")
}

func (tx *Transaction) ToProto() proto.Message {

	var vinArray []*transactionbasepb.TXInput
	for _, txin := range tx.Vin {
		vinArray = append(vinArray, txin.ToProto().(*transactionbasepb.TXInput))
	}

	var voutArray []*transactionbasepb.TXOutput
	for _, txout := range tx.Vout {
		voutArray = append(voutArray, txout.ToProto().(*transactionbasepb.TXOutput))
	}
	if tx.GasLimit == nil {
		tx.GasLimit = common.NewAmount(0)
	}
	if tx.GasPrice == nil {
		tx.GasPrice = common.NewAmount(0)
	}
	return &transactionpb.Transaction{
		Id:       tx.ID,
		Vin:      vinArray,
		Vout:     voutArray,
		Tip:      tx.Tip.Bytes(),
		GasLimit: tx.GasLimit.Bytes(),
		GasPrice: tx.GasPrice.Bytes(),
	}
}

func (tx *Transaction) FromProto(pb proto.Message) {
	tx.ID = pb.(*transactionpb.Transaction).GetId()
	tx.Tip = common.NewAmountFromBytes(pb.(*transactionpb.Transaction).GetTip())

	var vinArray []transaction_base.TXInput
	txin := transaction_base.TXInput{}
	for _, txinpb := range pb.(*transactionpb.Transaction).GetVin() {
		txin.FromProto(txinpb)
		vinArray = append(vinArray, txin)
	}
	tx.Vin = vinArray

	var voutArray []transaction_base.TXOutput
	txout := transaction_base.TXOutput{}
	for _, txoutpb := range pb.(*transactionpb.Transaction).GetVout() {
		txout.FromProto(txoutpb)
		voutArray = append(voutArray, txout)
	}
	tx.Vout = voutArray

	tx.GasLimit = common.NewAmountFromBytes(pb.(*transactionpb.Transaction).GetGasLimit())
	tx.GasPrice = common.NewAmountFromBytes(pb.(*transactionpb.Transaction).GetGasPrice())
}

func (tx *Transaction) GetSize() int {
	rawBytes, err := proto.Marshal(tx.ToProto())
	if err != nil {
		logger.Warn("Transaction: Transaction can not be marshalled!")
		return 0
	}
	return len(rawBytes)
}

// GasCountOfTxBase calculate the actual amount for a tx with data
func (ctx *ContractTx) GasCountOfTxBase() (*common.Amount, error) {
	txGas := MinGasCountPerTransaction
	if dataLen := ctx.DataLen(); dataLen > 0 {
		dataGas := common.NewAmount(uint64(dataLen)).Mul(GasCountPerByte)
		baseGas := txGas.Add(dataGas)
		txGas = baseGas
	}
	return txGas, nil
}

// DataLen return the length of payload
func (ctx *ContractTx) DataLen() int {
	return len([]byte(ctx.GetContract()))
}

// GetDefaultFromPubKeyHash returns the first from address public key hash
func (tx *Transaction) GetDefaultFromPubKeyHash() account.PubKeyHash {
	if tx.Vin == nil || len(tx.Vin) <= 0 {
		return nil
	}
	vin := tx.Vin[0]
	pubKeyHash, err := account.NewUserPubKeyHash(vin.PubKey)
	if err != nil {
		return nil
	}
	return pubKeyHash
}
