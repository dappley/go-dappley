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
// You should have received a copy of the GNU Gc
// eneral Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//
package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"time"

	"github.com/dappley/go-dappley/common"
	corepb "github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/crypto/sha3"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type BlockHeader struct {
	hash      Hash
	prevHash  Hash
	nonce     int64
	timestamp int64
	sign      Hash
	height    uint64
	producer  string
}

type Block struct {
	header       *BlockHeader
	transactions []*Transaction
}

type Hash []byte

func (h Hash) String() string {
	return hex.EncodeToString(h)
}

func (h Hash) Equals(nh Hash) bool {
	return bytes.Compare(h, nh) == 0
}

func NewBlock(txs []*Transaction, parent *Block, producer string) *Block {
	return NewBlockWithTimestamp(txs, parent, time.Now().Unix(), producer)
}

func NewBlockWithTimestamp(txs []*Transaction, parent *Block, timeStamp int64, producer string) *Block {

	var prevHash []byte
	var height uint64
	height = 1
	if parent != nil {
		prevHash = parent.GetHash()
		height = parent.GetHeight() + 1
	}

	if txs == nil {
		txs = []*Transaction{}
	}
	return &Block{
		header: &BlockHeader{
			hash:      []byte{},
			prevHash:  prevHash,
			nonce:     0,
			timestamp: timeStamp,
			sign:      nil,
			height:    height,
			producer:  producer,
		},
		transactions: txs,
	}
}

func (b *Block) BeIrreversible() {

}

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.transactions {
		txHashes = append(txHashes, tx.Hash())
	}
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}

func (b *Block) Serialize() []byte {
	rawBytes, err := proto.Marshal(b.ToProto())
	if err != nil {
		logger.WithError(err).Panic("Block: Cannot serialize block!")
	}
	logger.WithFields(logger.Fields{
		"size": len(rawBytes),
	}).Info("Block: Serialize Block!")
	return rawBytes
}

func Deserialize(d []byte) *Block {
	pb := &corepb.Block{}
	err := proto.Unmarshal(d, pb)
	if err != nil {
		logger.WithError(err).Panic("Block: Cannot deserialize block!")
	}
	block := &Block{}
	block.FromProto(pb)
	return block
}

func (b *Block) GetHeader() *BlockHeader {
	return b.header
}

func (b *Block) SetHash(hash Hash) {
	b.header.hash = hash
}

func (b *Block) GetHash() Hash {
	return b.header.hash
}

func (b *Block) GetSign() Hash {
	return b.header.sign
}

func (b *Block) GetHeight() uint64 {
	return b.header.height
}

func (b *Block) GetPrevHash() Hash {
	return b.header.prevHash
}

func (b *Block) SetNonce(nonce int64) {
	b.header.nonce = nonce
}

func (b *Block) GetNonce() int64 {
	return b.header.nonce
}

func (b *Block) GetTimestamp() int64 {
	return b.header.timestamp
}

func (b *Block) GetProducer() string {
	return b.header.producer
}

func (b *Block) GetTransactions() []*Transaction {
	return b.transactions
}

func (b *Block) ToProto() proto.Message {

	var txArray []*corepb.Transaction
	for _, tx := range b.transactions {
		txArray = append(txArray, tx.ToProto().(*corepb.Transaction))
	}

	return &corepb.Block{
		Header:       b.header.ToProto().(*corepb.BlockHeader),
		Transactions: txArray,
	}
}

func (b *Block) FromProto(pb proto.Message) {

	bh := BlockHeader{}
	bh.FromProto(pb.(*corepb.Block).GetHeader())
	b.header = &bh

	var txs []*Transaction

	for _, txpb := range pb.(*corepb.Block).GetTransactions() {
		tx := &Transaction{}
		tx.FromProto(txpb)
		txs = append(txs, tx)
	}
	b.transactions = txs
}

func (bh *BlockHeader) ToProto() proto.Message {
	return &corepb.BlockHeader{
		Hash:         bh.hash,
		PreviousHash: bh.prevHash,
		Nonce:        bh.nonce,
		Timestamp:    bh.timestamp,
		Signature:    bh.sign,
		Height:       bh.height,
		Producer:     bh.producer,
	}
}

func (bh *BlockHeader) FromProto(pb proto.Message) {
	if pb == nil {
		return
	}
	bh.hash = pb.(*corepb.BlockHeader).GetHash()
	bh.prevHash = pb.(*corepb.BlockHeader).GetPreviousHash()
	bh.nonce = pb.(*corepb.BlockHeader).GetNonce()
	bh.timestamp = pb.(*corepb.BlockHeader).GetTimestamp()
	bh.sign = pb.(*corepb.BlockHeader).GetSignature()
	bh.height = pb.(*corepb.BlockHeader).GetHeight()
	bh.producer = pb.(*corepb.BlockHeader).GetProducer()
}

func (b *Block) CalculateHash() Hash {
	return b.CalculateHashWithNonce(b.GetNonce())
}

func (b *Block) CalculateHashWithoutNonce() Hash {
	data := bytes.Join(
		[][]byte{
			b.GetPrevHash(),
			b.HashTransactions(),
			util.IntToHex(b.GetTimestamp()),
			[]byte(b.GetProducer()),
		},
		[]byte{},
	)

	hasher := sha3.New256()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func (b *Block) CalculateHashWithNonce(nonce int64) Hash {
	data := bytes.Join(
		[][]byte{
			b.GetPrevHash(),
			b.HashTransactions(),
			util.IntToHex(b.GetTimestamp()),
			//util.IntToHex(targetBits),
			util.IntToHex(nonce),
			[]byte(b.GetProducer()),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
	return hash[:]
}

func (b *Block) SignBlock(key string, data []byte) bool {
	if len(key) <= 0 {
		logger.Warn("Block: the key is too short for signature!")
		return false
	}
	privData, err := hex.DecodeString(key)

	if err != nil {
		logger.Warn("Block: cannot decode private key for signature!")
		return false
	}
	signature, err := secp256k1.Sign(data, privData)
	if err != nil {
		logger.WithError(err).Warn("Block: failed to calculate signature!")
		return false
	}

	b.header.sign = signature
	return true
}

func (b *Block) VerifyHash() bool {
	return bytes.Compare(b.GetHash(), b.CalculateHash()) == 0
}

func (b *Block) VerifyTransactions(utxoIndex *UTXOIndex, scState *ScState, manager ScEngineManager, parentBlk *Block) bool {
	if len(b.GetTransactions()) == 0 {
		logger.WithFields(logger.Fields{
			"hash": b.GetHash(),
		}).Debug("Block: there is no transaction to verify in this block.")
		return true
	}

	var rewardTX *Transaction
	var contractGeneratedTXs []*Transaction
	rewards := make(map[string]string)
	var allContractGeneratedTXs []*Transaction
	var scEngine ScEngine

	if manager != nil {
		scEngine = manager.CreateEngine()
		defer scEngine.DestroyEngine()
	}

L:
	for _, tx := range b.GetTransactions() {
		// Collect the contract-incurred transactions in this block
		if tx.IsRewardTx() {
			if rewardTX != nil {
				logger.Warn("Block: contains more than 1 reward transaction.")
				return false
			}
			rewardTX = tx
			utxoIndex.UpdateUtxo(tx)
			continue L
		}
		if tx.IsFromContract(utxoIndex) {
			contractGeneratedTXs = append(contractGeneratedTXs, tx)
			continue L
		}

		ctx := tx.ToContractTx()
		if ctx != nil {
			// Run the contract and collect generated transactions
			if scEngine == nil {
				logger.Warn("Block: smart contract cannot be verified.")
				logger.Debug("Block: is missing SCEngineManager when verifying transactions.")
				return false
			}

			prevUtxos, err := ctx.FindAllTxinsInUtxoPool(*utxoIndex)
			if err != nil {
				logger.WithError(err).WithFields(logger.Fields{
					"txid": hex.EncodeToString(ctx.ID),
				}).Warn("Transaction: cannot find vin while executing smart contract")
				return false
			}

			isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0

			utxoIndex.UpdateUtxo(tx)
			ctx.Execute(prevUtxos, isSCUTXO, *utxoIndex, scState, rewards, scEngine, b.GetHeight(), parentBlk)
			allContractGeneratedTXs = append(allContractGeneratedTXs, scEngine.GetGeneratedTXs()...)
		} else {
			// tx is a normal transactions
			if result, err := tx.Verify(utxoIndex, b.GetHeight()); !result {
				logger.Warn(err.Error())
				return false
			}
			utxoIndex.UpdateUtxo(tx)
		}
	}
	// Assert that any contract-incurred transactions matches the ones generated from contract execution
	if rewardTX != nil && !rewardTX.MatchRewards(rewards) {
		logger.Warn("Block: reward tx cannot be verified.")
		return false
	}
	if len(contractGeneratedTXs) > 0 && !verifyGeneratedTXs(utxoIndex, contractGeneratedTXs, allContractGeneratedTXs) {
		logger.Warn("Block: generated tx cannot be verified.")
		return false
	}
	utxoIndex.UpdateUtxoState(allContractGeneratedTXs)
	return true
}

// verifyGeneratedTXs verify that all transactions in candidates can be found in generatedTXs
func verifyGeneratedTXs(utxoIndex *UTXOIndex, candidates []*Transaction, generatedTXs []*Transaction) bool {
	// genTXBuckets stores description of txs grouped by concatenation of sender's and recipient's public key hashes
	genTXBuckets := make(map[string][][]*common.Amount)
	for _, genTX := range generatedTXs {
		sender, recipient, amount, tip, err := genTX.Describe(utxoIndex)
		if err != nil {
			continue
		}
		hashKey := sender.String() + recipient.String()
		genTXBuckets[hashKey] = append(genTXBuckets[hashKey], []*common.Amount{amount, tip})
	}
L:
	for _, tx := range candidates {
		sender, recipient, amount, tip, err := tx.Describe(utxoIndex)
		if err != nil {
			return false
		}
		hashKey := sender.String() + recipient.String()
		if genTXBuckets[hashKey] == nil {
			return false
		}
		for i, t := range genTXBuckets[hashKey] {
			// tx is verified if amount and tip matches
			if amount.Cmp(t[0]) == 0 && tip.Cmp(t[1]) == 0 {
				genTXBuckets[hashKey] = append(genTXBuckets[hashKey][:i], genTXBuckets[hashKey][i+1:]...)
				continue L
			}
		}
		return false
	}
	return true
}

func IsParentBlockHash(parentBlk, childBlk *Block) bool {
	if parentBlk == nil || childBlk == nil {
		return false
	}
	return reflect.DeepEqual(parentBlk.GetHash(), childBlk.GetPrevHash())
}

func IsHashEqual(h1 Hash, h2 Hash) bool {

	return reflect.DeepEqual(h1, h2)
}

func IsParentBlockHeight(parentBlk, childBlk *Block) bool {
	if parentBlk == nil || childBlk == nil {
		return false
	}
	return parentBlk.GetHeight() == childBlk.GetHeight()-1
}

func (b *Block) IsParentBlock(child *Block) bool {
	return IsParentBlockHash(b, child) && IsParentBlockHeight(b, child)
}

func (b *Block) Rollback(txPool *TransactionPool) {
	if b != nil {
		for _, tx := range b.GetTransactions() {
			if !tx.IsCoinbase() && !tx.IsRewardTx() {
				txPool.Push(*tx)
			}
		}
	}
}

func (b *Block) FindTransactionById(txid []byte) *Transaction {
	for _, tx := range b.transactions {
		if bytes.Compare(tx.ID, txid) == 0 {

			return tx
		}

	}
	return nil
}

func (b *Block) GetCoinbaseTransaction() *Transaction {
	//the coinbase transaction is usually placed at the end of all transactions
	for i := len(b.transactions) - 1; i >= 0; i-- {
		if b.transactions[i].IsCoinbase() {
			return b.transactions[i]
		}
	}
	return nil
}
