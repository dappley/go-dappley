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
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/dappley/go-dappley/core/utxo"
	"strings"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/transactionbase"
	transactionbasepb "github.com/dappley/go-dappley/core/transactionbase/pb"
	"github.com/dappley/go-dappley/crypto/byteutils"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

var Subsidy = common.NewAmount(10000000000)

const (
	ContractTxouputIndex = 0
	scheduleFuncName     = "dapp_schedule"
	SCDestroyAddress     = "dRxukNqeADQrAvnHD52BVNdGg6Bgmyuaw4"
)

var RewardTxData = []byte("Distribute X Rewards")
var GasRewardData = []byte("Miner Gas Rewards")
var GasChangeData = []byte("Unspent Gas Change")

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

type TxType int

const (
	TxTypeDefault      TxType = 0
	TxTypeNormal       TxType = 1
	TxTypeContract     TxType = 2
	TxTypeCoinbase     TxType = 3
	TxTypeGasReward    TxType = 4
	TxTypeGasChange    TxType = 5
	TxTypeReward       TxType = 6
	TxTypeContractSend TxType = 7
)

type Transaction struct {
	ID         []byte
	Vin        []transactionbase.TXInput
	Vout       []transactionbase.TXOutput
	Tip        *common.Amount
	GasLimit   *common.Amount
	GasPrice   *common.Amount
	CreateTime int64
	Type       TxType
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

//
func SetSubsidy(amount int)  {
	Subsidy = common.NewAmount(uint64(amount))
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

// Sign signs each input of a Transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error {
	txCopy := tx.TrimmedCopy(false)
	privData, err := secp256k1.FromECDSAPrivateKey(&privKey)
	if err != nil {
		logger.WithError(err).Error("Transaction: failed to get private key.")
		return err
	}

	for i, vin := range txCopy.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := vin.PubKey
		txCopy.Vin[i].PubKey = []byte(prevUtxos[i].PubKeyHash)
		txCopy.ID = txCopy.Hash()

		txCopy.Vin[i].PubKey = oldPubKey

		signature, err := secp256k1.Sign(txCopy.ID, privData)
		if err != nil {
			logger.WithError(err).Error("Transaction: failed to create a signature.")
			return err
		}

		tx.Vin[i].Signature = signature
	}
	return nil
}

// IsNormal returns true if tx a normal tx
func (tx *Transaction) IsNormal() bool {
	return tx.Type == TxTypeNormal
}

func (tx *Transaction) IsCoinbase() bool {
	return tx.Type == TxTypeCoinbase
}

// IsRewardTx returns if the transaction is about the step reward
func (tx *Transaction) IsRewardTx() bool {
	return tx.Type == TxTypeReward
}

// IsGasRewardTx returns if the transaction is about the gas reward to miner after smart contract execution
func (tx *Transaction) IsGasRewardTx() bool {
	return tx.Type == TxTypeGasReward
}

// IsGasChangeTx returns if the transaction is about the gas change to from address after smart contract execution
func (tx *Transaction) IsGasChangeTx() bool {
	return tx.Type == TxTypeGasChange
}

// IsContract returns true if the transaction deploys/executes a smart contract; false otherwise
func (tx *Transaction) IsContract() bool {
	return tx.Type == TxTypeContract
}

// IsContractSend returns true if the transaction is generated by contract execution; false otherwise
func (tx *Transaction) IsContractSend() bool {
	return tx.Type == TxTypeContractSend
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
	if tx.Type > TxTypeDefault {
		tempBytes = append(tempBytes, byteutils.FromInt32(int32(tx.Type))...)
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
	var inputs []transactionbase.TXInput
	var outputs []transactionbase.TXOutput
	var pubkey []byte

	for _, vin := range tx.Vin {
		if withSignature {
			pubkey = vin.PubKey
		} else {
			pubkey = nil
		}
		inputs = append(inputs, transactionbase.TXInput{vin.Txid, vin.Vout, nil, pubkey})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, transactionbase.TXOutput{vout.Value, vout.PubKeyHash, vout.Contract})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip, tx.GasLimit, tx.GasPrice, tx.CreateTime, tx.Type}

	return txCopy
}

func (tx *Transaction) DeepCopy() Transaction {
	var inputs []transactionbase.TXInput
	var outputs []transactionbase.TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, transactionbase.TXInput{vin.Txid, vin.Vout, vin.Signature, vin.PubKey})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, transactionbase.TXOutput{vout.Value, vout.PubKeyHash, vout.Contract})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip, tx.GasLimit, tx.GasPrice, tx.CreateTime, tx.Type}

	return txCopy
}

// VerifyID verifies if the transaction ID is the hash of the transaction
func (tx *Transaction) verifyID() (bool, error) {
	txCopy := tx.TrimmedCopy(true)
	if bytes.Equal(tx.ID, (&txCopy).Hash()) {
		return true, nil
	} else {
		return false, errors.New("Transaction: ID is invalid")
	}
}

// VerifyAmount verifies if the transaction has the correct vout value
func (tx *Transaction) verifyAmount(totalPrev *common.Amount, totalVoutValue *common.Amount) (bool, error) {
	//TotalVin amount must equal or greater than total vout
	if totalPrev.Cmp(totalVoutValue) < 0 {
		return false, errors.New("Transaction: amount is invalid")
	}
	sub, err := totalPrev.Sub(totalVoutValue)
	if err != nil {
		return false, err
	}
	if tx.GasLimit != nil {
		sub, err = sub.Sub(tx.GasLimit.Mul(tx.GasPrice))
		if err != nil {
			return false, errors.New("Transaction: GasLimit is invalid")
		}
	}
	if tx.Tip.Cmp(sub) != 0 {
		return false, errors.New("Transaction: tip is invalid")
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

func (tx *Transaction) MatchRewards(rewardStorage map[string]string) bool {

	if tx == nil {
		logger.Debug("Transaction: does not exist")
		return false
	}

	adaptedTx := NewTxAdapter(tx)
	if !adaptedTx.IsRewardTx() {
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
	lines = append(lines, fmt.Sprintf("     GasLimit %d:", tx.GasLimit))
	lines = append(lines, fmt.Sprintf("     GasPrice %d:", tx.GasPrice))
	lines = append(lines, fmt.Sprintf("     Type %d:", tx.Type))
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
		Type:     int32(tx.Type),
	}
}

func (tx *Transaction) FromProto(pb proto.Message) {
	tx.ID = pb.(*transactionpb.Transaction).GetId()
	tx.Tip = common.NewAmountFromBytes(pb.(*transactionpb.Transaction).GetTip())

	var vinArray []transactionbase.TXInput
	txin := transactionbase.TXInput{}
	for _, txinpb := range pb.(*transactionpb.Transaction).GetVin() {
		txin.FromProto(txinpb)
		vinArray = append(vinArray, txin)
	}
	tx.Vin = vinArray

	var voutArray []transactionbase.TXOutput
	txout := transactionbase.TXOutput{}
	for _, txoutpb := range pb.(*transactionpb.Transaction).GetVout() {
		txout.FromProto(txoutpb)
		voutArray = append(voutArray, txout)
	}
	tx.Vout = voutArray

	tx.GasLimit = common.NewAmountFromBytes(pb.(*transactionpb.Transaction).GetGasLimit())
	tx.GasPrice = common.NewAmountFromBytes(pb.(*transactionpb.Transaction).GetGasPrice())
	tx.Type = TxType(int(pb.(*transactionpb.Transaction).GetType()))
}

func (tx *Transaction) GetSize() int {
	rawBytes, err := proto.Marshal(tx.ToProto())
	if err != nil {
		logger.Warn("Transaction: Transaction can not be marshalled!")
		return 0
	}
	return len(rawBytes)
}

// GetDefaultFromPubKeyHash returns the first from address public key hash
func (tx *Transaction) GetDefaultFromTransactionAccount() *account.TransactionAccount {
	if tx.Vin == nil || len(tx.Vin) <= 0 {
		return account.NewContractTransactionAccount()
	}
	vin := tx.Vin[0]
	if ok, err := account.IsValidPubKey(vin.PubKey); !ok {
		logger.WithError(err).Warn("DPoS: cannot compute the public key hash!")
		return account.NewContractTransactionAccount()
	}

	ta := account.NewTransactionAccountByPubKey(vin.PubKey)

	return ta
}

//CalculateUtxoSum calculates the total amount of all input utxos
func CalculateUtxoSum(utxos []*utxo.UTXO) *common.Amount {
	sum := common.NewAmount(0)
	for _, utxo := range utxos {
		sum = sum.Add(utxo.Value)
	}
	return sum
}

//CalculateChange calculates the change
func CalculateChange(input, amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount) (*common.Amount, error) {
	change, err := input.Sub(amount)
	if err != nil {
		return nil, ErrInsufficientFund
	}

	change, err = change.Sub(tip)
	if err != nil {
		return nil, ErrInsufficientFund
	}
	change, err = change.Sub(gasLimit.Mul(gasPrice))
	if err != nil {
		return nil, ErrInsufficientFund
	}
	if change.Cmp(common.NewAmount(0)) < 0 {
		return nil, ErrInsufficientFund
	}
	return change, nil
}

func (tx *Transaction) VerifySignatures(prevUtxos []*utxo.UTXO) (bool, error) {
	txCopy := tx.TrimmedCopy(false)

	for i, vin := range tx.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := txCopy.Vin[i].PubKey
		txCopy.Vin[i].PubKey = []byte(prevUtxos[i].PubKeyHash)
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[i].PubKey = oldPubKey

		originPub := make([]byte, 1+len(vin.PubKey))
		originPub[0] = 4 // uncompressed point
		copy(originPub[1:], vin.PubKey)

		if vin.Signature == nil || len(vin.Signature) == 0 {
			return false, errors.New("Transaction: Signatures is empty")
		}

		verifyResult, err := secp256k1.Verify(txCopy.ID, vin.Signature, originPub)

		if err != nil || verifyResult == false {
			return false, errors.New("Transaction: Signatures is invalid")
		}
	}

	return true, nil
}

//verifyPublicKeyHash verifies if the public key in Vin is the original key for the public
//key hash in utxo
func (tx *Transaction) VerifyPublicKeyHash(prevUtxos []*utxo.UTXO) (bool, error) {

	for i, vin := range tx.Vin {
		if prevUtxos[i].PubKeyHash == nil {
			logger.Error("Transaction: previous transaction is not correct.")
			return false, errors.New("Transaction: prevUtxos not found")
		}

		isContract, err := prevUtxos[i].PubKeyHash.IsContract()
		if err != nil {
			return false, err
		}
		//if the utxo belongs to a Contract, the utxo is not verified through
		//public key hash. It will be verified through consensus
		if isContract {
			continue
		}
		if ok, err := account.IsValidPubKey(vin.PubKey); !ok {
			logger.WithError(err).Warn("DPoS: cannot compute the public key hash!")
			return false, err
		}

		ta := account.NewTransactionAccountByPubKey(vin.PubKey)

		if !bytes.Equal([]byte(ta.GetPubKeyHash()), []byte(prevUtxos[i].PubKeyHash)) {
			return false, errors.New("Transaction: ID is invalid")
		}
	}
	return true, nil
}

func (tx *Transaction) Verify(prevUtxos []*utxo.UTXO) error {
	if prevUtxos == nil {
		return errors.New("Transaction: prevUtxos not found")
	}
	result, err := tx.verifyID()
	if !result {
		return err
	}

	result, err = tx.VerifyPublicKeyHash(prevUtxos)
	if !result {
		return err
	}

	totalPrev := CalculateUtxoSum(prevUtxos)
	totalVoutValue, ok := tx.CalculateTotalVoutValue()
	if !ok {
		return errors.New("Transaction: vout is invalid")
	}
	result, err = tx.verifyAmount(totalPrev, totalVoutValue)
	if !result {
		return err
	}
	result, err = tx.VerifySignatures(prevUtxos)
	if !result {
		return err
	}

	return nil
}
