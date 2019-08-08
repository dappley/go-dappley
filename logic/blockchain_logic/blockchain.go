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

package blockchain_logic

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/block_logic"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction_base"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/jinzhu/copier"
	logger "github.com/sirupsen/logrus"
)

var tipKey = []byte("tailBlockHash")
var libKey = []byte("lastIrreversibleBlockHash")

var (
	ErrBlockDoesNotExist       = errors.New("block does not exist")
	ErrPrevHashVerifyFailed    = errors.New("prevhash verify failed")
	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrTransactionVerifyFailed = errors.New("transaction verification failed")
	ErrRewardTxVerifyFailed    = errors.New("Verify reward transaction failed")
	ErrProducerNotEnough       = errors.New("producer number is less than ConsensusSize")
	// DefaultGasPrice default price of per gas
	DefaultGasPrice uint64 = 1
)

type Blockchain struct {
	bc           *blockchain.Blockchain
	db           storage.Storage
	utxoCache    *core.UTXOCache
	consensus    core.Consensus
	txPool       *core.TransactionPool
	scManager    core.ScEngineManager
	eventManager *core.EventManager
	blkSizeLimit int
	mutex        *sync.Mutex
}

// CreateBlockchain creates a new blockchain db
func CreateBlockchain(address account.Address, db storage.Storage, consensus core.Consensus, txPool *core.TransactionPool, scManager core.ScEngineManager, blkSizeLimit int) *Blockchain {
	genesis := NewGenesisBlock(address, core.Subsidy)
	bc := &Blockchain{
		blockchain.NewBlockchain(genesis.GetHash(), genesis.GetHash()),
		db,
		utxo.NewUTXOCache(db),
		consensus,
		txPool,
		scManager,
		core.NewEventManager(),
		blkSizeLimit,
		&sync.Mutex{},
	}
	utxoIndex := core.NewUTXOIndex(bc.GetUtxoCache())
	utxoIndex.UpdateUtxoState(genesis.GetTransactions())
	scState := core.NewScState()
	err := bc.AddBlockContextToTail(&core.BlockContext{Block: genesis, Lib: genesis, UtxoIndex: utxoIndex, State: scState})
	if err != nil {
		logger.Panic("CreateBlockchain: failed to add genesis block!")
	}
	return bc
}

func GetBlockchain(db storage.Storage, consensus core.Consensus, txPool *core.TransactionPool, scManager core.ScEngineManager, blkSizeLimit int) (*Blockchain, error) {
	var tip []byte
	tip, err := db.Get(tipKey)
	if err != nil {
		return nil, err
	}
	lib, err := db.Get(libKey)
	if err != nil {
		return nil, err
	}

	bc := &Blockchain{
		blockchain.NewBlockchain(tip, lib),
		db,
		utxo.NewUTXOCache(db),
		consensus,
		txPool,
		scManager,
		core.NewEventManager(),
		blkSizeLimit,
		&sync.Mutex{},
	}
	return bc, nil
}

func (bc *Blockchain) GetDb() storage.Storage {
	return bc.db
}

func (bc *Blockchain) GetUtxoCache() *utxo.UTXOCache {
	return bc.utxoCache
}

func (bc *Blockchain) GetTailBlockHash() hash.Hash {
	return bc.bc.GetTailBlockHash()
}

func (bc *Blockchain) GetLIBHash() hash.Hash {
	return bc.bc.GetLIBHash()
}

func (bc *Blockchain) GetSCManager() core.ScEngineManager {
	return bc.scManager
}

func (bc *Blockchain) GetConsensus() core.Consensus {
	return bc.consensus
}

func (bc *Blockchain) GetTxPool() *core.TransactionPool {
	return bc.txPool
}

func (bc *Blockchain) GetEventManager() *core.EventManager {
	return bc.eventManager
}

func (bc *Blockchain) SetBlockSizeLimit(limit int) {
	bc.blkSizeLimit = limit
}

func (bc *Blockchain) GetBlockSizeLimit() int {
	return bc.blkSizeLimit
}

func (bc *Blockchain) GetTailBlock() (*block.Block, error) {
	hash := bc.GetTailBlockHash()
	return bc.GetBlockByHash(hash)
}

func (bc *Blockchain) GetLIB() (*block.Block, error) {
	hash := bc.GetLIBHash()
	return bc.GetBlockByHash(hash)
}

func (bc *Blockchain) GetMaxHeight() uint64 {
	block, err := bc.GetTailBlock()
	if err != nil {
		return 0
	}
	return block.GetHeight()
}

func (bc *Blockchain) GetLIBHeight() uint64 {
	block, err := bc.GetLIB()
	if err != nil {
		return 0
	}
	return block.GetHeight()
}

func (bc *Blockchain) GetBlockByHash(hash hash.Hash) (*block.Block, error) {
	rawBytes, err := bc.db.Get(hash)
	if err != nil {
		return nil, ErrBlockDoesNotExist
	}
	return block.Deserialize(rawBytes), nil
}

func (bc *Blockchain) GetBlockByHeight(height uint64) (*block.Block, error) {
	hash, err := bc.db.Get(util.UintToHex(height))
	if err != nil {
		return nil, ErrBlockDoesNotExist
	}

	return bc.GetBlockByHash(hash)
}

func (bc *Blockchain) SetTailBlockHash(tailBlockHash hash.Hash) {
	bc.SetTailBlockHash(tailBlockHash)
}

func (bc *Blockchain) SetConsensus(consensus core.Consensus) {
	bc.consensus = consensus
}

func (bc *Blockchain) SetState(state blockchain.BlockchainState) {
	bc.bc.SetState(state)
}

func (bc *Blockchain) GetState() blockchain.BlockchainState {
	return bc.bc.GetState()
}

func (bc *Blockchain) AddBlockContextToTail(ctx *core.BlockContext) error {
	// Atomically set tail block hash and update UTXO index in db
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	tailBlockHash := bc.GetTailBlockHash()
	if ctx.Block.GetHeight() != 0 && bytes.Compare(ctx.Block.GetPrevHash(), tailBlockHash) != 0 {
		return ErrPrevHashVerifyFailed
	}

	blockLogger := logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	})

	bcTemp := bc.deepCopy()
	tailBlk, _ := bc.GetTailBlock()

	bcTemp.db.EnableBatch()
	defer bcTemp.db.DisableBatch()

	if ctx.Lib != nil {
		err := bcTemp.SetLIBHash(ctx.Lib.GetHash())
		if err != nil {
			blockLogger.Error("Blockchain: failed to set lib hash!")
			return err
		}
	}

	err := bcTemp.setTailBlockHash(ctx.Block.GetHash())
	if err != nil {
		blockLogger.Error("Blockchain: failed to set tail block hash!")
		return err
	}

	numTxBeforeExe := bc.GetTxPool().GetNumOfTxInPool()

	bcTemp.runScheduleEvents(ctx, tailBlk)
	err = ctx.UtxoIndex.Save()
	if err != nil {
		blockLogger.Warn("Blockchain: failed to save utxo to database.")
		return err
	}

	//Remove transactions in current transaction pool
	bcTemp.GetTxPool().CleanUpMinedTxs(ctx.Block.GetTransactions())
	bcTemp.GetTxPool().ResetPendingTransactions()
	err = bcTemp.GetTxPool().SaveToDatabase(bc.db)

	if err != nil {
		blockLogger.Warn("Blockchain: failed to save txpool to database.")
		return err
	}

	logger.WithFields(logger.Fields{
		"num_txs_before_add_block":    numTxBeforeExe,
		"num_txs_after_update_txpool": bc.GetTxPool().GetNumOfTxInPool(),
	}).Info("Blockchain : update tx pool")

	err = bcTemp.AddBlockToDb(ctx.Block)
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
		poolsize = bc.txPool.GetNumOfTxInPool()
	}

	blockLogger.WithFields(logger.Fields{
		"numOfTx":  len(ctx.Block.GetTransactions()),
		"poolSize": poolsize,
	}).Info("Blockchain: added a new block to tail.")

	return nil
}

func (bc *Blockchain) runScheduleEvents(ctx *core.BlockContext, parentBlk *block.Block) error {
	if parentBlk == nil {
		//if the current block is genesis block. do not run smart contract
		return nil
	}

	if bc.scManager == nil {
		return nil
	}

	bc.scManager.RunScheduledEvents(ctx.UtxoIndex.GetContractUtxos(), ctx.State, ctx.Block.GetHeight(), parentBlk.GetTimestamp())
	bc.eventManager.Trigger(ctx.State.GetEvents())
	return nil
}

func (bc *Blockchain) FindTXOutput(in transaction_base.TXInput) (transaction_base.TXOutput, error) {
	db := bc.db
	vout, err := core.GetTxOutput(in, db)
	if err != nil {
		return transaction_base.TXOutput{}, err
	}
	return vout, err
}

func (bc *Blockchain) Iterator() *Blockchain {
	return &Blockchain{
		blockchain.NewBlockchain(bc.GetTailBlockHash(), bc.GetLIBHash()),
		bc.db,
		bc.utxoCache,
		bc.consensus,
		nil,
		nil,
		nil,
		bc.blkSizeLimit,
		bc.mutex,
	}
}

func (bc *Blockchain) Next() (*block.Block, error) {
	var blk *block.Block

	encodedBlock, err := bc.db.Get(bc.GetTailBlockHash())
	if err != nil {
		return nil, err
	}

	blk = block.Deserialize(encodedBlock)

	bc.bc.SetTailBlockHash(blk.GetPrevHash())

	return blk, nil
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
func (bc *Blockchain) AddBlockToDb(blk *block.Block) error {

	err := bc.db.Put(blk.GetHash(), blk.Serialize())
	if err != nil {
		logger.WithError(err).Warn("Blockchain: failed to add blk to database!")
		return err
	}

	err = bc.db.Put(util.UintToHex(blk.GetHeight()), blk.GetHash())
	if err != nil {
		logger.WithError(err).Warn("Blockchain: failed to index the blk by blk height in database!")
		return err
	}
	// add transaction journals
	for _, tx := range blk.GetTransactions() {
		err = core.PutTxJournal(*tx, bc.db)
		if err != nil {
			logger.WithError(err).Warn("Blockchain: failed to add blk transaction journals into database!")
			return err
		}
	}
	return nil
}

func (bc *Blockchain) IsHigherThanBlockchain(block *block.Block) bool {
	return block.GetHeight() > bc.GetMaxHeight()
}

func (bc *Blockchain) IsInBlockchain(hash hash.Hash) bool {
	_, err := bc.GetBlockByHash(hash)
	return err == nil
}

//rollback the blockchain to a block with the targetHash
func (bc *Blockchain) Rollback(targetHash hash.Hash, utxo *core.UTXOIndex, scState *core.ScState) bool {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if !bc.IsInBlockchain(targetHash) {
		return false
	}
	parentblockHash := bc.GetTailBlockHash()
	//if is child of tail, skip rollback
	if block_logic.IsHashEqual(parentblockHash, targetHash) {
		return true
	}

	//keep rolling back blocks until the block with the input hash
	for bytes.Compare(parentblockHash, targetHash) != 0 {

		block, err := bc.GetBlockByHash(parentblockHash)
		logger.WithFields(logger.Fields{
			"height": block.GetHeight(),
			"hash":   parentblockHash.String(),
		}).Info("Blockchain: is about to rollback the block...")
		if err != nil {
			return false
		}
		parentblockHash = block.GetPrevHash()

		for _, tx := range block.GetTransactions() {
			if !tx.IsCoinbase() && !tx.IsRewardTx() && !tx.IsGasRewardTx() && !tx.IsGasChangeTx() {
				bc.txPool.Rollback(*tx)
			}
		}
	}

	bc.db.EnableBatch()
	defer bc.db.DisableBatch()

	err := bc.setTailBlockHash(parentblockHash)
	if err != nil {
		logger.Error("Blockchain: failed to set tail block hash during rollback!")
		return false
	}

	bc.txPool.SaveToDatabase(bc.db)

	utxo.Save()
	scState.SaveToDatabase(bc.db)
	bc.db.Flush()

	return true
}

func (bc *Blockchain) setTailBlockHash(hash hash.Hash) error {
	err := bc.db.Put(tipKey, hash)
	if err != nil {
		return err
	}
	bc.bc.SetTailBlockHash(hash)
	return nil
}

func (bc *Blockchain) deepCopy() *Blockchain {
	newCopy := &Blockchain{}
	copier.Copy(newCopy, bc)
	return newCopy
}

func (bc *Blockchain) SetLIBHash(hash hash.Hash) error {
	err := bc.db.Put(libKey, hash)
	if err != nil {
		return err
	}
	bc.bc.SetLIBHash(hash)
	return nil
}

func (bc *Blockchain) IsLIB(blk *block.Block) bool {
	blkFromDb, err := bc.GetBlockByHash(blk.GetHash())
	if err != nil {
		logger.Error("Blockchain:get block by hash from blockchain error: ", err)
		return false
	}
	if blkFromDb == nil {
		logger.Error("Blockchain:blk is not exist in blockchain")
		return false
	}

	lib, _ := bc.GetLIB()

	if lib.GetHeight() >= blkFromDb.GetHeight() {
		return true
	}
	return false
}

// GasPrice returns gas price in current blockchain
func (bc *Blockchain) GasPrice() uint64 {
	return DefaultGasPrice
}
