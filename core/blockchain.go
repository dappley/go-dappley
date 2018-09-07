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
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

var tipKey = []byte("1")

const BlockPoolMaxSize = 100

var (
	ErrBlockDoesNotExist   = errors.New("ERROR: Block does not exist in blockchain")
	ErrTransactionNotFound = errors.New("ERROR: Transaction not found")
)

type Blockchain struct {
	tailBlockHash []byte
	db            storage.Storage
	blockPool     *BlockPool
	consensus     Consensus
	txPool        *TransactionPool
}

// CreateBlockchain creates a new blockchain db
func CreateBlockchain(address Address, db storage.Storage, consensus Consensus) *Blockchain {
	genesis := NewGenesisBlock(address.Address)
	bc := &Blockchain{
		genesis.GetHash(),
		db,
		NewBlockPool(BlockPoolMaxSize),
		consensus,
		NewTransactionPool(),
	}
	bc.blockPool.SetBlockchain(bc)
	bc.AddBlockToTail(genesis)
	return bc
}

func GetBlockchain(db storage.Storage, consensus Consensus) (*Blockchain, error) {
	var tip []byte
	tip, err := db.Get(tipKey)
	if err != nil {
		return nil, err
	}
	bc := &Blockchain{
		tip,
		db,
		NewBlockPool(BlockPoolMaxSize),
		consensus,
		NewTransactionPool(), //TODO: Need to retrieve transaction pool from db
	}
	bc.blockPool.SetBlockchain(bc)
	return bc, nil
}

func (bc *Blockchain) GetDb() storage.Storage {
	return bc.db
}

func (bc *Blockchain) GetTailBlockHash() Hash {
	return bc.tailBlockHash
}

func (bc *Blockchain) GetBlockPool() *BlockPool {
	return bc.blockPool
}

func (bc *Blockchain) GetConsensus() Consensus {
	return bc.consensus
}

func (bc *Blockchain) GetTxPool() *TransactionPool {
	return bc.txPool
}

func (bc *Blockchain) GetTailBlock() (*Block, error) {
	hash := bc.GetTailBlockHash()
	return bc.GetBlockByHash(hash)
}

func (bc *Blockchain) GetMaxHeight() uint64 {
	blk, err := bc.GetTailBlock()
	if err != nil {
		return 0
	}
	return blk.GetHeight()
}

func (bc *Blockchain) GetBlockByHash(hash Hash) (*Block, error) {
	rawBytes, err := bc.db.Get(hash)
	if err != nil {
		return nil, ErrBlockDoesNotExist
	}
	return Deserialize(rawBytes), nil
}

func (bc *Blockchain) SetTailBlockHash(tailBlockHash Hash) {
	bc.tailBlockHash = tailBlockHash
}

func (bc *Blockchain) SetConsensus(consensus Consensus) {
	bc.consensus = consensus
}

func (bc *Blockchain) AddBlockToTail(newBlock *Block) {
	logger.Info("Blockchain: Updated A New Block! Height:", newBlock.GetHeight(), " Hash:", hex.EncodeToString(newBlock.GetHash()))
	bc.AddBlockToDatabase(newBlock)
	bc.setTailBlockHash(newBlock.GetHash())
	newBlock.UpdateUtxoIndexAfterNewBlock(UtxoMapKey, bc.db)
}

//TODO: optimize performance
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block, err := bci.Next()
		if err != nil {
			return Transaction{}, err
		}

		for _, tx := range block.GetTransactions() {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}

	return Transaction{}, ErrTransactionNotFound
}

func (bc *Blockchain) FindTransactionFromIndexBlock(txID []byte, blockId []byte) (Transaction, error) {
	println("as")

	bci := bc.Iterator()

	for {
		block, err := bci.NextFromIndex(blockId)
		if err != nil {
			println("1")
			return Transaction{}, err
		}

		for _, tx := range block.GetTransactions() {
			if bytes.Compare(tx.ID, txID) == 0 {
				return *tx, nil
			}
		}

		if len(block.GetPrevHash()) == 0 {
			println("2")
			break
		}
	}

	return Transaction{}, ErrTransactionNotFound
}

//TODO: optimize performance
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) ([]Transaction, error) {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block, err := bci.Next()
		if err != nil {
		}

		for _, tx := range block.GetTransactions() {
			txID := hex.EncodeToString(tx.ID)

		Outputs: //TODO
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}

	return unspentTXs, nil
}

func (bc *Blockchain) FindUTXO(pubKeyHash []byte) ([]TXOutput, error) {
	var UTXOs []TXOutput
	unspentTransactions, err := bc.FindUnspentTransactions(pubKeyHash)
	if err != nil {
		return nil, err
	}

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs, nil
}

func (bc *Blockchain) Iterator() *Blockchain {
	return &Blockchain{bc.tailBlockHash, bc.db, nil, bc.consensus, nil}
}

func (bc *Blockchain) Next() (*Block, error) {
	var block *Block

	encodedBlock, err := bc.db.Get(bc.tailBlockHash)
	if err != nil {
		return nil, err
	}

	block = Deserialize(encodedBlock)

	bc.tailBlockHash = block.GetPrevHash()

	return block, nil
}

func (bc *Blockchain) NextFromIndex(indexHash []byte) (*Block, error) {
	var block *Block

	encodedBlock, err := bc.db.Get(indexHash)
	if err != nil {
		return nil, err
	}

	block = Deserialize(encodedBlock)

	bc.tailBlockHash = block.GetPrevHash()
	println(bc.tailBlockHash)
	return block, nil
}

func (bc *Blockchain) String() string {
	var buffer bytes.Buffer

	bci := bc.Iterator()
	for {
		block, err := bci.Next()
		if err != nil {
			fmt.Println(err)
		}

		buffer.WriteString(fmt.Sprintf("============ Block %x ============\n", block.GetHash()))
		buffer.WriteString(fmt.Sprintf("Height: %d\n", block.GetHeight()))
		buffer.WriteString(fmt.Sprintf("Prev. block: %x\n", block.GetPrevHash()))
		for _, tx := range block.GetTransactions() {
			buffer.WriteString(tx.String())
		}
		buffer.WriteString(fmt.Sprintf("\n\n"))

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}
	return buffer.String()
}

//record the new block in the database
func (bc *Blockchain) AddBlockToDatabase(newBlock *Block) {
	bc.db.Put(newBlock.GetHash(), newBlock.Serialize())
}

func (bc *Blockchain) IsHigherThanBlockchain(blk *Block) bool {
	return blk.GetHeight() > bc.GetMaxHeight()
}

func (bc *Blockchain) IsInBlockchain(hash Hash) bool {
	_, err := bc.GetBlockByHash(hash)
	return err == nil
}

func (bc *Blockchain) MergeFork() {

	//find parent block
	forkHeadBlock := bc.GetBlockPool().GetForkPoolHeadBlk()
	if forkHeadBlock == nil {
		return
	}
	forkParentHash := forkHeadBlock.GetPrevHash()
	if !bc.IsInBlockchain(forkParentHash) {
		return
	}

	//verify transactions in the fork
	utxo, err := bc.GetUtxoStateAtBlockHash(bc.db, forkParentHash)
	if err != nil {
		logger.Warn(err)
	}
	if !bc.GetBlockPool().VerifyTransactions(utxo) {
		return
	}

	bc.Rollback(forkParentHash)

	//add all blocks in fork from head to tail
	bc.concatenateForkToBlockchain()

	logger.Debug("Merged Fork!!")
}

func (bc *Blockchain) concatenateForkToBlockchain() {
	if bc.GetBlockPool().forkPoolLen() > 0 {
		for i := bc.GetBlockPool().forkPoolLen() - 1; i >= 0; i-- {
			bc.AddBlockToTail(bc.GetBlockPool().forkPool[i])
			//Remove transactions in current transaction pool
			bc.GetTxPool().RemoveMultipleTransactions(bc.GetBlockPool().forkPool[i].GetTransactions())
		}
	}
	bc.GetBlockPool().ResetForkPool()
}

//rollback the blockchain to a block with the targetHash
func (bc *Blockchain) Rollback(targetHash Hash) bool {

	if !bc.IsInBlockchain(targetHash) {
		return false
	}

	parentBlkHash := bc.GetTailBlockHash()

	//keep rolling back blocks until the block with the input hash
loop:
	for {
		if bytes.Compare(parentBlkHash, targetHash) == 0 {
			break loop
		}
		blk, err := bc.GetBlockByHash(parentBlkHash)

		if err != nil {
			return false
		}
		parentBlkHash = blk.GetPrevHash()
		blk.Rollback(bc.txPool)
	}
	bc.setTailBlockHash(parentBlkHash)

	return true
}

func (bc *Blockchain) setTailBlockHash(hash Hash) {
	bc.db.Put(tipKey, hash)
	bc.tailBlockHash = hash
}
