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
// You should have received a copy of the GNU Gc
// eneral Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//
package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
	logger "github.com/sirupsen/logrus"

	"math/big"
	"reflect"

	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/util"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"encoding/hex"
)


type BlockHeader struct {
	hash      Hash
	prevHash  Hash
	nonce     int64
	timestamp int64
	sign Hash
	height       uint64
}

type Block struct {
	header       *BlockHeader
	transactions []*Transaction
}

type Hash []byte

func NewBlock(transactions []*Transaction, parent *Block) *Block {

	var prevHash []byte
	var height uint64
	height = 1
	if parent != nil {
		prevHash = parent.GetHash()
		height = parent.GetHeight() + 1
	}

	if transactions == nil {
		transactions = []*Transaction{}
	}
	return &Block{
		header: &BlockHeader{
			hash:      []byte{},
			prevHash:  prevHash,
			nonce:     0,
			timestamp: time.Now().Unix(),
			sign: nil,
			height: height,
		},
		transactions: transactions,
	}
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
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	bs := &BlockStream{
		Header: &BlockHeaderStream{
			Hash:      b.header.hash,
			PrevHash:  b.header.prevHash,
			Nonce:     b.header.nonce,
			Timestamp: b.header.timestamp,
			Sign: b.header.sign,
			Height: b.header.height,
		},
		Transactions: b.transactions,
	}

	err := encoder.Encode(bs)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

func Deserialize(d []byte) *Block {
	var bs BlockStream
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&bs)
	if err != nil {
		log.Panic(err)
	}
	if bs.Header.Hash == nil {
		bs.Header.Hash = Hash{}
	}
	if bs.Header.PrevHash == nil {
		bs.Header.PrevHash = Hash{}
	}
	if bs.Transactions == nil {
		bs.Transactions = []*Transaction{}
	}
	return &Block{
		header: &BlockHeader{
			hash:      bs.Header.Hash,
			prevHash:  bs.Header.PrevHash,
			nonce:     bs.Header.Nonce,
			timestamp: bs.Header.Timestamp,
			sign: bs.Header.Sign,
			height:	   bs.Header.Height,
		},
		transactions: bs.Transactions,
	}
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
	bh.FromProto(pb.(*corepb.Block).Header)
	b.header = &bh

	var txs []*Transaction

	for _, txpb := range pb.(*corepb.Block).Transactions {
		tx := &Transaction{}
		tx.FromProto(txpb)
		txs = append(txs, tx)
	}
	b.transactions = txs
	}


func (bh *BlockHeader) ToProto() proto.Message {
	return &corepb.BlockHeader{
		Hash:      bh.hash,
		Prevhash:  bh.prevHash,
		Nonce:     bh.nonce,
		Timestamp: bh.timestamp,
		Sign: bh.sign,
		Height: bh.height,

	}
}

func (bh *BlockHeader) FromProto(pb proto.Message) {
	bh.hash = pb.(*corepb.BlockHeader).Hash
	bh.prevHash = pb.(*corepb.BlockHeader).Prevhash
	bh.nonce = pb.(*corepb.BlockHeader).Nonce
	bh.timestamp = pb.(*corepb.BlockHeader).Timestamp
	bh.sign =  pb.(*corepb.BlockHeader).Sign
	bh.height =  pb.(*corepb.BlockHeader).Height
}

func (b *Block) CalculateHash() Hash {
	return b.CalculateHashWithoutNonce()
}

func (b *Block) CalculateHashWithoutNonce() Hash {
	var hashInt big.Int
	data := bytes.Join(
		[][]byte{
			b.GetPrevHash(),
			b.HashTransactions(),
			util.IntToHex(b.GetTimestamp()),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	return hashInt.Bytes()
}

func (b *Block) CalculateHashWithNonce(nonce int64) Hash {
	var hashInt big.Int
	data := bytes.Join(
		[][]byte{
			b.GetPrevHash(),
			b.HashTransactions(),
			util.IntToHex(b.GetTimestamp()),
			//util.IntToHex(targetBits),
			util.IntToHex(nonce),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	return hashInt.Bytes()
}

func (b *Block) SignBlock(key string, data []byte) bool {
	if len(key) <= 0 {
		logger.Warn("Block: key length not enough for signature!")
		return false
	}
	privData, err := hex.DecodeString(key)

	if err != nil {
		logger.Warn("Block: private key decode error for signature!")
		return false
	}
	signature, err := secp256k1.Sign(data, privData)
	if err != nil {
		logger.Warn("Block: signature caculation error!")
		return false
	}

	b.header.sign = signature
	return true
}

func (b *Block) VerifyHash() bool {
	return bytes.Compare(b.GetHash(), b.CalculateHash()) == 0
}

func (b *Block) VerifyTransactions(utxo UtxoIndex) bool {
	for _, tx := range b.GetTransactions() {
		if !tx.Verify(utxo) {
			return false
		}
	}
	return true
}

func IsParentBlockHash(parentBlk, childBlk *Block) bool{
	if parentBlk == nil || childBlk == nil{
		return false
	}
	return reflect.DeepEqual(parentBlk.GetHash(), childBlk.GetPrevHash())
}

func IsParentBlockHeight(parentBlk, childBlk *Block) bool{
	if parentBlk == nil || childBlk == nil{
		return false
	}
	return parentBlk.GetHeight() == childBlk.GetHeight()-1
}

func IsParentBlock(parentBlk, childBlk *Block) bool{
	return IsParentBlockHash(parentBlk, childBlk) && IsParentBlockHeight(parentBlk, childBlk)
}

func (b *Block) Rollback(txPool *TransactionPool){
	if b!= nil {
		for _,tx := range b.GetTransactions(){
			if !tx.IsCoinbase() {
				txPool.Transactions.StructPush(*tx)
			}
		}
	}
}

func (b *Block) FindTransactionById(txid []byte) *Transaction{
	for _, tx := range b.transactions {
		if bytes.Compare(tx.ID, txid) == 0 {

			return tx
		}

	}
	return nil
}


func (b *Block) GetCoinbaseTransaction() *Transaction{
	//the coinbase transaction is usually placed at the end of all transactions
	for i:=len(b.transactions)-1;i>=0;i--{
		if b.transactions[i].IsCoinbase(){
			return b.transactions[i]
		}
	}
	return nil
}
