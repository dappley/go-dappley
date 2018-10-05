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
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"math/big"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/storage"
	"github.com/gogo/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

var subsidy = common.NewAmount(10)

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

// Hash returns the hash of the Transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// Sign signs each input of a Transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) error {
	if tx.IsCoinbase() {
		logger.Warning("Coinbase transaction could not be signed")
		return nil
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			logger.Error("Previous transaction is invalid")
			return ErrTXInputNotFound
		}
		if vin.Vout >= len(prevTXs[hex.EncodeToString(vin.Txid)].Vout) {
			logger.Error("Input of the transaction not found in previous transactions")
			return ErrTXInputNotFound
		}
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		privData, err := secp256k1.FromECDSAPrivateKey(&privKey)
		if err != nil {
			logger.Error("ERROR: Get private key failed", err)
			return err
		}

		signature, err := secp256k1.Sign(txCopy.ID, privData)
		if err != nil {
			logger.Error("ERROR: Sign transaction.Id failed", err)
			return err
		}

		tx.Vin[inID].Signature = signature
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

	prevUtxos := tx.FindAllTxinsInUtxoPool(utxo)
	if prevUtxos == nil {
		return false
	}
	return tx.VerifySignatures(prevUtxos)
}

// VerifySignatures verifies signatures of Transaction inputs
func (tx *Transaction) VerifySignatures(prevTXOutputs map[string]TXOutput) bool {

	var verifyResult bool
	var error1 error

	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXOutputs[hex.EncodeToString(vin.Txid)].PubKeyHash == nil {
			logger.Error("Previous transaction is invalid")
			return false
		}
	}

	txCopy := tx.TrimmedCopy()
	//	curve := elliptic.P256()
	curve := secp256k1.S256()

	for inID, vin := range tx.Vin {
		prevTXOut := prevTXOutputs[hex.EncodeToString(vin.Txid)]
		vinPubKeyHash, _ := HashPubKey(vin.PubKey)
		if bytes.Compare(prevTXOut.PubKeyHash, vinPubKeyHash) != 0 {
			logger.Error("Vout Vin public key mismatch")
			return false
		}
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTXOut.PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil
		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		originPub, err := secp256k1.FromECDSAPublicKey(&rawPubKey)
		if err != nil {
			logger.Error(err)
			return false
		}

		verifyResult, error1 = secp256k1.Verify(txCopy.ID, vin.Signature, originPub)

		if error1 != nil || verifyResult == false {
			logger.Error(error1)
			return false
		}
	}

	return true
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to, data string, blockHeight uint64) Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}
	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := TXInput{nil, -1, bh, []byte(data)}
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}, 0}
	tx.ID = tx.Hash()

	return tx
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(db storage.Storage, from, to Address, amount *common.Amount, senderKeyPair KeyPair, bc *Blockchain, tip uint64) (Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput
	var validOutputs []*UTXO

	pubKeyHash, _ := HashPubKey(senderKeyPair.PublicKey)
	sum := common.NewAmount(0)
	senderUTXOs := LoadUTXOIndex(db).GetUTXOsByPubKeyHash(pubKeyHash)

	if len(senderUTXOs) < 1 {
		return Transaction{}, ErrInsufficientFund
	}
	for _, v := range senderUTXOs {
		sum = sum.Add(v.Value)
		validOutputs = append(validOutputs, v)
		if sum.Cmp(amount) >= 0 {
			break
		}
	}

	if sum.Cmp(amount) < 0 { // TODO: add tips
		return Transaction{}, ErrInsufficientFund
	}

	// Build a list of inputs
	for _, out := range validOutputs {
		input := TXInput{out.Txid, out.TxIndex, nil, senderKeyPair.PublicKey}
		inputs = append(inputs, input)

	}
	// Build a list of outputs
	outputs = append(outputs, *NewTXOutput(amount, to.Address))
	if sum.Cmp(amount) > 0 {
		change, err := sum.Sub(amount)
		if err != nil {
			logger.Panic(err)
		}
		outputs = append(outputs, *NewTXOutput(change, from.Address))
	}

	tx := Transaction{nil, inputs, outputs, tip}
	tx.ID = tx.Hash()
	prevTXs := tx.GetPrevTransactions(bc)
	err := tx.Sign(senderKeyPair.PrivateKey, prevTXs)
	if err != nil {
		logger.Error(err)
		return Transaction{}, err
	}

	return tx, nil
}

func (tx *Transaction) GetPrevTransactions(bc *Blockchain) map[string]Transaction {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			logger.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return prevTXs
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

//Find the transaction in a utxo pool. Returns true only if all Vins are found in the utxo pool
func (tx *Transaction) FindAllTxinsInUtxoPool(utxoPool UTXOIndex) map[string]TXOutput {
	res := make(map[string]TXOutput)
	for _, vin := range tx.Vin {
		utxo := utxoPool.FindUTXO(vin.Txid, vin.Vout)
		if utxo == nil {
			return nil
		}
		txout := TXOutput{utxo.Value, utxo.PubKeyHash}
		res[hex.EncodeToString(vin.Txid)] = txout
	}
	return res
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
