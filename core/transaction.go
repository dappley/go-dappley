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

package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/crypto/byteutils"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/gogo/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

var subsidy = common.NewAmount(10)
var enableAddBalanceTest = true

var (
	ErrInsufficientFund = errors.New("transaction: the balance is insufficient")
	ErrInvalidAmount    = errors.New("transaction: amount is invalid (must be > 0)")
	ErrTXInputNotFound  = errors.New("transaction: transaction input not found")
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
	Tip  uint64
}

type TxIndex struct {
	BlockId    []byte
	BlockIndex int
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1 && len(tx.Vout) == 1
}

// Serialize returns a serialized Transaction
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		logger.Panic(err)
	}

	return encoded.Bytes()
}

//GetToHashBytes Get bytes for hash
func (tx *Transaction) GetToHashBytes() []byte {
	var bytes []byte

	for _, vin := range tx.Vin {
		bytes = append(bytes, vin.Txid...)
		// int size may differ from differnt platform
		bytes = append(bytes, byteutils.FromInt32(int32(vin.Vout))...)
		bytes = append(bytes, vin.PubKey...)
		bytes = append(bytes, vin.Signature...)
	}

	for _, vout := range tx.Vout {
		bytes = append(bytes, vout.Value.Bytes()...)
		bytes = append(bytes, vout.PubKeyHash...)
	}

	bytes = append(bytes, byteutils.FromUint64(tx.Tip)...)
	return bytes
}

// Hash returns the hash of the Transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	hash = sha256.Sum256(tx.GetToHashBytes())

	return hash[:]
}

// Sign signs each input of a Transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevUtxos []*UTXO) error {
	if tx.IsCoinbase() {
		logger.Warning("Coinbase transaction could not be signed")
		return nil
	}

	txCopy := tx.TrimmedCopy()
	privData, err := secp256k1.FromECDSAPrivateKey(&privKey)
	if err != nil {
		logger.Error("ERROR: Get private key failed", err)
		return err
	}

	for i, vin := range txCopy.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := vin.PubKey
		txCopy.Vin[i].PubKey = prevUtxos[i].PubKeyHash
		txCopy.ID = txCopy.Hash()

		txCopy.Vin[i].PubKey = oldPubKey

		signature, err := secp256k1.Sign(txCopy.ID, privData)
		if err != nil {
			logger.Error("ERROR: Sign transaction.Id failed", err)
			return err
		}

		tx.Vin[i].Signature = signature
	}
	return nil
}

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip}

	return txCopy
}

// Verify ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func (tx *Transaction) Verify(utxo UTXOIndex, blockHeight uint64) bool {

	if tx.IsCoinbase() {
		if tx.Vout[0].Value.Cmp(subsidy) != 0 {
			return false
		}
		bh := binary.BigEndian.Uint64(tx.Vin[0].Signature)
		if blockHeight != bh {
			return false
		}
		return true
	}

	prevUtxos, err := tx.FindAllTxinsInUtxoPool(utxo)
	if err != nil {
		logger.Errorf("ERROR: %v", err)
		return false
	}

	//TODO  Remove the enableAddBalanceTest flag
	if !enableAddBalanceTest && tx.verifyAmount(prevUtxos) && tx.verifyTip(prevUtxos) == false {
		logger.Error("ERROR: Transaction amount is invalid")
		return false
	}

	return tx.verifySignatures(prevUtxos)
}

//verifyTip verifies if the transaction has the correct tip
func (tx *Transaction) verifyTip(prevUtxos []*UTXO) bool{
	sum := calculateUtxoSum(prevUtxos)
	var err error
	for _,vout := range tx.Vout{
		sum, err = sum.Sub(vout.Value)
		if err!=nil {
			return false
		}
	}
	return tx.Tip == sum.Uint64()
}

func (tx *Transaction) verifySignatures(prevUtxos []*UTXO) bool {
	for _, utxo := range prevUtxos {
		if utxo.PubKeyHash == nil {
			logger.Error("ERROR: Previous transaction is not correct")
			return false
		}
	}

	txCopy := tx.TrimmedCopy()

	for i, vin := range tx.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := txCopy.Vin[i].PubKey
		txCopy.Vin[i].PubKey = prevUtxos[i].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[i].PubKey = oldPubKey

		originPub := make([]byte, 1+len(vin.PubKey))
		originPub[0] = 4 // uncompressed point
		copy(originPub[1:], vin.PubKey)

		verifyResult, error1 := secp256k1.Verify(txCopy.ID, vin.Signature, originPub)

		if error1 != nil || verifyResult == false {
			logger.Errorf("Error: Verify sign failed %v", error1)
			return false
		}
	}

	return true
}

func (tx *Transaction) verifyAmount(prevTXs []*UTXO) bool {
	var totalVin, totalVout common.Amount
	for _, utxo := range prevTXs {
		totalVin = *totalVin.Add(utxo.Value)
	}

	for _, vout := range tx.Vout {
		totalVout = *totalVout.Add(vout.Value)
	}

	//TotalVin amount must equal or greater than total vout
	return totalVin.Cmp(&totalVout) >= 0
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to, data string, blockHeight uint64, tip *common.Amount) Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}
	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := TXInput{nil, -1, bh, []byte(data)}
	txout := NewTXOutput(subsidy.Add(tip), to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}, 0}
	tx.ID = tx.Hash()

	return tx
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(utxos []*UTXO, from, to Address, amount *common.Amount, senderKeyPair KeyPair, tip *common.Amount) (Transaction, error) {

	sum := calculateUtxoSum(utxos)
	change,err := calculateChange(sum, amount, tip)
	if err!= nil {
		return Transaction{}, err
	}

	tx := Transaction{
		nil,
		prepareInputLists(utxos, senderKeyPair.PublicKey),
		prepareOutputLists(from, to, amount, change),
		tip.Uint64()}
	tx.ID = tx.Hash()

	err = tx.Sign(senderKeyPair.PrivateKey, utxos)
	if err != nil {
		return Transaction{}, err
	}

	return tx,nil
}

//for add balance
func NewUTXOTransactionforAddBalance(to Address, amount *common.Amount) (Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput

	// Validate amount
	if amount.Validate() != nil || amount.IsZero() {
		return Transaction{}, ErrInvalidAmount
	}

	// Build a list of outputs
	outputs = append(outputs, *NewTXOutput(amount, to.Address))

	tx := Transaction{nil, inputs, outputs, 0}
	tx.ID = tx.Hash()

	return tx, nil
}

//FindAllTxinsInUtxoPool Find the transaction in a utxo pool. Returns true only if all Vins are found in the utxo pool
func (tx *Transaction) FindAllTxinsInUtxoPool(utxoPool UTXOIndex) ([]*UTXO, error) {
	var res []*UTXO
	for _, vin := range tx.Vin {
		pubKeyHash, err := HashPubKey(vin.PubKey)
		if err != nil {
			return nil, ErrTXInputNotFound
		}
		utxo := utxoPool.FindUTXOByVin(pubKeyHash, vin.Txid, vin.Vout)
		if utxo == nil {
			return nil, ErrTXInputNotFound
		}
		res = append(res, utxo)
	}
	return res, nil
}

// String returns a human-readable representation of a transaction
func (tx Transaction) String() string {
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
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}
	lines = append(lines, "\n")

	return strings.Join(lines, "\n")
}

func (tx *Transaction) ToProto() proto.Message {

	var vinArray []*corepb.TXInput
	for _, txin := range tx.Vin {
		vinArray = append(vinArray, txin.ToProto().(*corepb.TXInput))
	}

	var voutArray []*corepb.TXOutput
	for _, txout := range tx.Vout {
		voutArray = append(voutArray, txout.ToProto().(*corepb.TXOutput))
	}

	return &corepb.Transaction{
		ID:   tx.ID,
		Vin:  vinArray,
		Vout: voutArray,
		Tip:  tx.Tip,
	}
}

func (tx *Transaction) FromProto(pb proto.Message) {
	tx.ID = pb.(*corepb.Transaction).ID
	tx.Tip = pb.(*corepb.Transaction).Tip

	var vinArray []TXInput
	txin := TXInput{}
	for _, txinpb := range pb.(*corepb.Transaction).Vin {
		txin.FromProto(txinpb)
		vinArray = append(vinArray, txin)
	}
	tx.Vin = vinArray

	var voutArray []TXOutput
	txout := TXOutput{}
	for _, txoutpb := range pb.(*corepb.Transaction).Vout {
		txout.FromProto(txoutpb)
		voutArray = append(voutArray, txout)
	}
	tx.Vout = voutArray
}


//calculateChange calculates the change
func calculateChange(input, amount, tip *common.Amount) (*common.Amount, error){
	change, err := input.Sub(amount)
	if err != nil {
		return nil, ErrInsufficientFund
	}

	change, err = change.Sub(tip)
	if err != nil {
		return nil, ErrInsufficientFund
	}
	return change, nil
}

//prepareInputLists prepares a list of txinputs for a new transaction
func prepareInputLists(utxos []*UTXO, publicKey []byte) []TXInput{
	var inputs []TXInput

	// Build a list of inputs
	for _, utxo := range utxos {
		input := TXInput{utxo.Txid, utxo.TxIndex, nil, publicKey}
		inputs = append(inputs, input)
	}

	return inputs
}

//calculateUtxoSum calculates the total amount of all input utxos
func calculateUtxoSum(utxos []*UTXO) *common.Amount{
	sum := common.NewAmount(0)
	for _, utxo := range utxos {
		sum = sum.Add(utxo.Value)
	}
	return sum
}

//preapreOutPutLists prepares a list of txoutputs for a new transaction
func prepareOutputLists(from,to Address, amount *common.Amount, change *common.Amount) []TXOutput{

	var outputs []TXOutput
	// Build a list of outputs
	outputs = append(outputs, *NewTXOutput(amount, to.Address))
	outputs = append(outputs, *NewTXOutput(change, from.Address))
	return outputs
}