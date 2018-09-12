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
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
	"github.com/jinzhu/copier"
)

const UtxoMapKey = "utxo"
const UtxoForkMapKey = "utxoFork"

//map of key: wallet public key hash, value: serialized map
type utxoIndex map[string][]UTXOutputStored

type UTXOutputStored struct {
	Value      *common.Amount
	PubKeyHash []byte
	Txid       []byte
	TxIndex    int
}

func DeserializeUTXO(d []byte) *utxoIndex {
	var utxo utxoIndex
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&utxo)
	if err != nil {
		fmt.Printf("%+v\n", err.Error())
	}
	return &utxo
}

func (utxo *utxoIndex) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(utxo)
	if err != nil {
		log.Panic(err)
	}
	return encoded.Bytes()
}

func GetAddressUTXOs(mapkey string, pubkeyHash []byte, db storage.Storage) []UTXOutputStored {
	umap := GetStoredUtxoMap(db, mapkey)
	return umap[string(pubkeyHash)]
}

func GetStoredUtxoMap(db storage.Storage, mapkey string) utxoIndex {
	res, err := db.Get([]byte(mapkey))

	if err != nil && strings.Contains(err.Error(), "Key is invalid") {
		return utxoIndex{}
	}
	umap := DeserializeUTXO(res)
	return *umap
}

func initIndex() utxoIndex {
	ins := map[string][]UTXOutputStored{}
	return ins
}

func (blk Block) UpdateUtxoIndexAfterNewBlock(mapkey string, db storage.Storage) {
	//remove expended outputs
	blk.ConsumeSpendableOutputsAfterNewBlock(mapkey, db)
	//add new outputs
	blk.AddSpendableOutputsAfterNewBlock(mapkey, db)
}

func (blk Block) AddSpendableOutputsAfterNewBlock(mapkey string, db storage.Storage) {
	utxoIndex := GetStoredUtxoMap(db, mapkey)

	if len(utxoIndex) == 0 {
		utxoIndex = initIndex()
	}
	for _, tx := range blk.transactions {
		for index, vout := range tx.Vout {
			if utxoIndex[string(vout.PubKeyHash)] == nil {
				utxoIndex[string(vout.PubKeyHash)] = []UTXOutputStored{}
			}
			utxoIndex[string(vout.PubKeyHash)] = append(utxoIndex[string(vout.PubKeyHash)], UTXOutputStored{vout.Value, vout.PubKeyHash, tx.ID, index})
		}
	}
	db.Put([]byte(UtxoMapKey), utxoIndex.Serialize())
}

func (blk Block) ConsumeSpendableOutputsAfterNewBlock(mapkey string, db storage.Storage) {
	utxoIndex := GetStoredUtxoMap(db, mapkey)
	for _, txs := range blk.transactions {
		if txs.IsCoinbase() {
			continue
		}

		for _, vin := range txs.Vin {
			// Key in utxoIndex is PubkeyHash
			spentOutputTxId, txIndex := vin.Txid, vin.Vout
			pubKeyHash, _ := HashPubKey(vin.PubKey)
			userUtxos := utxoIndex[string(pubKeyHash)]
			if (len(userUtxos)) > 0 {
				for index, userUtxo := range userUtxos {
					if userUtxo.TxIndex == txIndex && bytes.Compare(userUtxo.Txid, spentOutputTxId) == 0 {
						userUtxos = append(userUtxos[:index], userUtxos[index+1:]...)
					}
				}
				//write to index
				utxoIndex[string(pubKeyHash)] = userUtxos
			}
		}
	}
	utxoIndex.SetUtxoPoolInDb(db)
}

func (utxo *utxoIndex) FindUtxoByTxinput(txin TXInput) *UTXOutputStored {
	for _, utxoArray := range *utxo {
		for _, u := range utxoArray {
			if bytes.Compare(u.Txid, txin.Txid) == 0 && u.TxIndex == txin.Vout {
				return &u
			}
		}
	}
	return nil
}

//doesnt save to db
func (utxo utxoIndex) RevertTxUtxos(blk Block, bc Blockchain, db storage.Storage) {

	for _, tx := range blk.GetTransactions() {
		err1 := utxo.RemoveTxUtxosFromUtxoPool(*tx, db)
		if err1 != nil {
			log.Panic(err1)
		}

		if tx.IsCoinbase() {
			continue
		}

		err2 := utxo.AddBackTxOutputToUtxoPool(*tx, db, blk, bc)
		if err2 != nil {
			log.Panic(err2)
		}
	}
}

func (utxo utxoIndex) RemoveTxUtxosFromUtxoPool(txs Transaction, db storage.Storage) error {

	for outIndex, out := range txs.Vout {
		userUtxos := utxo[string(out.PubKeyHash)]

	Stud:
		for index, userUtxo := range userUtxos {
			if userUtxo.TxIndex == outIndex && bytes.Compare(userUtxo.Txid, txs.ID) == 0 {
				//remove utxo from index
				userUtxos = append(userUtxos[:index], userUtxos[index+1:]...)
				break Stud
			} else {
				log.Panic("Address given has no utxos in index")
			}
		}
		utxo[string(out.PubKeyHash)] = userUtxos
	}
	return nil
}

func (utxo utxoIndex) AddBackTxOutputToUtxoPool(tx Transaction, db storage.Storage, blk Block, bc Blockchain) error {
	for _, vin := range tx.Vin {
		vout, voutIndex, err := getTXOFromTxIn(vin, blk.GetHash(), bc)
		if err == nil {
			pubKeyHash, _ := HashPubKey(vin.PubKey)
			utxo[string(vout.PubKeyHash)] = append(utxo[string(vout.PubKeyHash)], UTXOutputStored{vout.Value, pubKeyHash, tx.ID, voutIndex})
		} else {
			panic(err)
		}
	}
	return nil
}

//set utxopool
func (utxo utxoIndex) SetUtxoPoolInDb(db storage.Storage) {
	db.Put([]byte(UtxoMapKey), utxo.Serialize())
}

//block is passed in because i cant statically call FindTransactionById

func getTXOFromTxIn(in TXInput, blkStartIndex []byte, bc Blockchain) (TXOutput, int, error) {
	tx, err := bc.FindTransaction(in.Txid)
	if err != nil {
		return TXOutput{}, 0, errors.New("txInput refers to nonexisting tx")
	}
	return tx.Vout[in.Vout], in.Vout, nil
}

func (utxo utxoIndex) DeepCopy(db storage.Storage) utxoIndex {
	utxocopy := utxoIndex{}
	copier.Copy(&utxo, &utxocopy)
	if len(utxocopy) == 0 {
		utxocopy = initIndex()
	}
	return utxocopy
}

//input db and block hash, output utxoindex state @block hash block
func (bc Blockchain) GetUtxoStateAtBlockHash(db storage.Storage, hash []byte) (utxoIndex, error) {
	index := GetStoredUtxoMap(db, UtxoMapKey)
	deepCopy := index.DeepCopy(db)
	bci := bc.Iterator()

	for {
		block, err := bci.Next()

		if bytes.Compare(block.GetHash(), hash) == 0 {
			break
		}

		if err != nil {
			return utxoIndex{}, err
		}

		if len(block.GetPrevHash()) == 0 {
			return utxoIndex{}, ErrBlockDoesNotExist
		}

		deepCopy.RevertTxUtxos(*block, bc, db)

	}

	return deepCopy, nil
}
