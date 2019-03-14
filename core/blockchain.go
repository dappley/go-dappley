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

package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/jinzhu/copier"
	logger "github.com/sirupsen/logrus"
)

var tipKey = []byte("tailBlockHash")

const LengthForBlockToBeConsideredHistory = 100

var (
	ErrBlockDoesNotExist       = errors.New("block does not exist")
	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrTransactionVerifyFailed = errors.New("transaction verify failed")
	ErrRewardTxVerifyFailed    = errors.New("Verify reward transaction failed")
)

type BlockchainState int

const (
	BlockchainInit BlockchainState = iota
	BlockchainDownloading
	BlockchainSync
	BlockchainReady
)

type Blockchain struct {
	tailBlockHash []byte
	db            storage.Storage
	consensus     Consensus
	txPool        *TransactionPool
	scManager     ScEngineManager
	state         BlockchainState
	eventManager  *EventManager
}

// CreateBlockchain creates a new blockchain db
func CreateBlockchain(address Address, db storage.Storage, consensus Consensus, transactionPoolLimit uint32, scManager ScEngineManager) *Blockchain {
	genesis := NewGenesisBlock(address)
	bc := &Blockchain{
		genesis.GetHash(),
		db,
		consensus,
		NewTransactionPool(transactionPoolLimit),
		scManager,
		BlockchainReady,
		NewEventManager(),
	}
	bc.txPool.LoadFromDatabase(bc.db)
	err := bc.AddBlockToTail(genesis)
	if err != nil {
		logger.Panic("CreateBlockchain: failed to add genesis block!")
	}
	return bc
}

func GetBlockchain(db storage.Storage, consensus Consensus, transactionPoolLimit uint32, scManager ScEngineManager) (*Blockchain, error) {
	var tip []byte
	tip, err := db.Get(tipKey)
	if err != nil {
		return nil, err
	}

	bc := &Blockchain{
		tip,
		db,
		consensus,
		NewTransactionPool(transactionPoolLimit),
		scManager,
		BlockchainReady,
		NewEventManager(),
	}
	bc.txPool.LoadFromDatabase(bc.db)
	return bc, nil
}

func (bc *Blockchain) GetDb() storage.Storage {
	return bc.db
}

func (bc *Blockchain) GetTailBlockHash() Hash {
	return bc.tailBlockHash
}

func (bc *Blockchain) GetSCManager() ScEngineManager {
	return bc.scManager
}

func (bc *Blockchain) GetConsensus() Consensus {
	return bc.consensus
}

func (bc *Blockchain) GetTxPool() *TransactionPool {
	return bc.txPool
}

func (bc *Blockchain) GetEventManager() *EventManager {
	return bc.eventManager
}

func (bc *Blockchain) GetTailBlock() (*Block, error) {
	hash := bc.GetTailBlockHash()
	return bc.GetBlockByHash(hash)
}

func (bc *Blockchain) GetMaxHeight() uint64 {
	block, err := bc.GetTailBlock()
	if err != nil {
		return 0
	}
	return block.GetHeight()
}

func (bc *Blockchain) GetBlockByHash(hash Hash) (*Block, error) {
	rawBytes, err := bc.db.Get(hash)
	if err != nil {
		return nil, ErrBlockDoesNotExist
	}
	return Deserialize(rawBytes), nil
}

func (bc *Blockchain) GetBlockByHeight(height uint64) (*Block, error) {
	hash, err := bc.db.Get(util.UintToHex(height))
	if err != nil {
		return nil, ErrBlockDoesNotExist
	}

	return bc.GetBlockByHash(hash)
}

func (bc *Blockchain) SetTailBlockHash(tailBlockHash Hash) {
	bc.tailBlockHash = tailBlockHash
}

func (bc *Blockchain) SetConsensus(consensus Consensus) {
	bc.consensus = consensus
}

func (bc *Blockchain) SetState(state BlockchainState) {
	bc.state = state
}

func (bc *Blockchain) GetState() BlockchainState {
	return bc.state
}

func (bc *Blockchain) AddBlockToTail(block *Block) error {
	blockLogger := logger.WithFields(logger.Fields{
		"height": block.GetHeight(),
		"hash":   hex.EncodeToString(block.GetHash()),
	})

	// Atomically set tail block hash and update UTXO index in db
	bcTemp := bc.deepCopy()

	tailBlk, _ := bc.GetTailBlock()

	bcTemp.db.EnableBatch()
	defer bcTemp.db.DisableBatch()

	err := bcTemp.setTailBlockHash(block.GetHash())
	if err != nil {
		blockLogger.Error("Blockchain: failed to set tail block hash!")
		return err
	}

	numTxBeforeExe := len(bc.GetTxPool().GetTransactions())

	utxoIndex := LoadUTXOIndex(bc.db)
	tempUtxo := utxoIndex.DeepCopy()
	bcTemp.executeTransactionsAndUpdateScState(tempUtxo, block, tailBlk)
	utxoIndex.UpdateUtxoState(block.GetTransactions())
	err = utxoIndex.Save(bc.db)

	if err != nil {
		blockLogger.Warn("Blockchain: failed to save utxo to database.")
		return err
	}

	numTxAfterExe := len(bc.GetTxPool().GetTransactions())
	//Remove transactions in current transaction pool
	bc.GetTxPool().CheckAndRemoveTransactions(block.GetTransactions())
	err = bc.GetTxPool().SaveToDatabase(bc.db)

	if err != nil {
		blockLogger.Warn("Blockchain: failed to save txpool to database.")
		return err
	}

	logger.WithFields(logger.Fields{
		"num_txs_before_sc_exe":       numTxBeforeExe,
		"num_txs_after_sc_exe":        numTxAfterExe,
		"num_txs_after_update_txpool": len(bc.GetTxPool().GetTransactions()),
	}).Info("Blockchain : update tx pool")

	err = bcTemp.AddBlockToDb(block)
	if err != nil {
		blockLogger.Warn("Blockchain: failed to add block to database.")
		return err
	}

	// Flush batch changes to storage
	err = bcTemp.db.Flush()
	if err != nil {
		blockLogger.Error("Blockchain: failed to update tail block hash and UTXO index!")
		return err
	}

	// Assign changes to receiver
	*bc = *bcTemp

	poolsize := 0
	if bc.txPool != nil {
		poolsize = len(bc.txPool.GetTransactions())
	}

	blockLogger.WithFields(logger.Fields{
		"numOfTx":  len(block.GetTransactions()),
		"poolSize": poolsize,
	}).Info("Blockchain: added a new block to tail.")

	return nil
}

func (bc *Blockchain) executeTransactionsAndUpdateScState(utxoIndex *UTXOIndex, currBlock *Block, parentBlk *Block) error {

	if parentBlk == nil {
		//if the current block is genesis block. do not run smart contract
		return nil
	}

	if bc.scManager == nil {
		return nil
	}

	scState := NewScState()
	scState.LoadFromDatabase(bc.db)

	scEngine := bc.scManager.CreateEngine()
	defer scEngine.DestroyEngine()

	rewards := make(map[string]string)
	var rewardTX *Transaction

	for _, tx := range currBlock.GetTransactions() {
		genTxs := tx.Execute(*utxoIndex, scState, rewards, scEngine, currBlock.GetHeight(), parentBlk)
		for _, gtx := range genTxs {
			bc.GetTxPool().Push(*gtx)
		}

		if tx.IsRewardTx() {
			rewardTX = tx
		}

		utxoIndex.UpdateUtxo(tx)
	}

	if rewardTX != nil && !rewardTX.MatchRewards(rewards) {
		logger.Warn("Block: reward tx cannot be verified.")
		return ErrRewardTxVerifyFailed
	}

	bc.scManager.RunScheduledEvents(utxoIndex.GetContractUtxos(), scState, currBlock.GetHeight(), parentBlk.GetTimestamp())

	bc.eventManager.Trigger(scState.GetEvents())

	err := scState.Save(bc.db, currBlock.GetHash())
	if err != nil {
		return err
	}

	return nil
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
	bci := bc.Iterator()

	for {
		block, err := bci.NextFromIndex(blockId)
		if err != nil {
			return Transaction{}, err
		}

		for _, tx := range block.GetTransactions() {
			if bytes.Compare(tx.ID, txID) == 0 {
				return *tx, nil
			}
		}

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}

	return Transaction{}, ErrTransactionNotFound
}

func (bc *Blockchain) Iterator() *Blockchain {
	return &Blockchain{bc.tailBlockHash, bc.db, bc.consensus, nil, nil, BlockchainInit, nil}
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
			logger.Error(err)
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

//AddBlockToDb record the new block in the database
func (bc *Blockchain) AddBlockToDb(block *Block) error {

	err := bc.db.Put(block.GetHash(), block.Serialize())
	if err != nil {
		logger.WithError(err).Warn("Blockchain: failed to add block to database!")
		return err
	}

	err = bc.db.Put(util.UintToHex(block.GetHeight()), block.GetHash())
	if err != nil {
		logger.WithError(err).Warn("Blockchain: failed to index the block by block height in database!")
		return err
	}

	return nil
}

func (bc *Blockchain) IsHigherThanBlockchain(block *Block) bool {
	return block.GetHeight() > bc.GetMaxHeight()
}

func (bc *Blockchain) IsInBlockchain(hash Hash) bool {
	_, err := bc.GetBlockByHash(hash)
	return err == nil
}

func (bc *Blockchain) addBlocksToTail(blocks []*Block) {
	if len(blocks) > 0 {
		for i := len(blocks) - 1; i >= 0; i-- {
			err := bc.AddBlockToTail(blocks[i])
			if err != nil {
				logger.WithError(err).Error("Blockchain: failed to add block to tail while concatenating fork!")
				return
			}
		}
	}
}

//rollback the blockchain to a block with the targetHash
func (bc *Blockchain) Rollback(targetHash Hash, utxo *UTXOIndex) bool {

	if !bc.IsInBlockchain(targetHash) {
		return false
	}
	parentblockHash := bc.GetTailBlockHash()
	//if is child of tail, skip rollback
	if IsHashEqual(parentblockHash, targetHash) {
		return true
	}

	//keep rolling back blocks until the block with the input hash
loop:
	for {
		if bytes.Compare(parentblockHash, targetHash) == 0 {
			break loop
		}
		block, err := bc.GetBlockByHash(parentblockHash)
		logger.WithFields(logger.Fields{
			"height": block.GetHeight(),
			"hash":   hex.EncodeToString(parentblockHash),
		}).Info("Blockchain: is about to rollback the block...")
		if err != nil {
			return false
		}
		parentblockHash = block.GetPrevHash()
		block.Rollback(bc.txPool)
	}

	bc.db.EnableBatch()
	defer bc.db.DisableBatch()

	err := bc.setTailBlockHash(parentblockHash)
	if err != nil {
		logger.Error("Blockchain: failed to set tail block hash during rollback!")
		return false
	}
	utxo.Save(bc.db)
	bc.db.Flush()

	return true
}

func (bc *Blockchain) setTailBlockHash(hash Hash) error {
	err := bc.db.Put(tipKey, hash)
	if err != nil {
		return err
	}
	bc.tailBlockHash = hash
	return nil
}

func (bc *Blockchain) deepCopy() *Blockchain {
	newCopy := &Blockchain{}
	copier.Copy(&newCopy, &bc)
	return newCopy
}
