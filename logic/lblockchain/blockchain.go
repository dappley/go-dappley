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

package lblockchain

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/logic/transactionpool"

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/lblock"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	logger "github.com/sirupsen/logrus"
)

var tailBlockHash = []byte("tailBlockHash")

var UtxoSaveHash = []byte("utxoSaved")

var (
	ErrBlockDoesNotExist       = errors.New("block does not exist in db")
	ErrBlockDoesNotFound       = errors.New("the block does not found after lib")
	ErrPrevHashVerifyFailed    = errors.New("prevhash verify failed")
	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrTransactionVerifyFailed = errors.New("transaction verification failed")
	ErrRewardTxVerifyFailed    = errors.New("Verify reward transaction failed")
	ErrProducerNotEnough       = errors.New("producer number is less than ConsensusSize")
	// DefaultGasPrice default price of per gas
	DefaultGasPrice uint64 = 1
)

type Blockchain struct {
	bc           blockchain.Blockchain
	db           storage.Storage
	utxoCache    *utxo.UTXOCache
	libPolicy    LIBPolicy
	txPool       *transactionpool.TransactionPool
	eventManager *scState.EventManager
	blkSizeLimit int
	mutex        *sync.Mutex
}

// CreateBlockchain creates a new blockchain db
func CreateBlockchain(address account.Address, db storage.Storage, libPolicy LIBPolicy, txPool *transactionpool.TransactionPool, blkSizeLimit int) *Blockchain {
	genesis := NewGenesisBlock(address, transaction.Subsidy)
	bc := &Blockchain{
		blockchain.NewBlockchain(genesis.GetHash(), genesis.GetHash()),
		db,
		utxo.NewUTXOCache(db),
		libPolicy,
		txPool,
		scState.NewEventManager(),
		blkSizeLimit,
		&sync.Mutex{},
	}
	utxoIndex := lutxo.NewUTXOIndex(bc.GetUtxoCache())
	utxoIndex.UpdateUtxos(genesis.GetTransactions())
	scState := scState.NewScState(bc.GetUtxoCache())
	err := bc.AddBlockContextToTail(&BlockContext{Block: genesis, UtxoIndex: utxoIndex, State: scState})
	if err != nil {
		logger.Panic("CreateBlockchain: failed to add genesis block!")
	}
	return bc
}

func GetBlockchain(db storage.Storage, libPolicy LIBPolicy, txPool *transactionpool.TransactionPool, blkSizeLimit int) (*Blockchain, error) {
	var tip []byte
	tip, err := db.Get(tailBlockHash)
	if err != nil {
		logger.Warn("get tailBlockHash failed")
		return nil, err
	}

	bc := &Blockchain{
		blockchain.NewBlockchain(tip, []byte{}),
		db,
		utxo.NewUTXOCache(db),
		libPolicy,
		txPool,
		scState.NewEventManager(),
		blkSizeLimit,
		&sync.Mutex{},
	}

	lib, err := bc.getLIB(bc.GetMaxHeight())
	if err != nil {
		logger.Warn("getLIB failed")
		return nil, err
	}
	bc.SetLIBHash(lib)

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

func (bc *Blockchain) GetTxPool() *transactionpool.TransactionPool {
	return bc.txPool
}

func (bc *Blockchain) GetEventManager() *scState.EventManager {
	return bc.eventManager
}

func (bc *Blockchain) GetUpdatedUTXOIndex() (*lutxo.UTXOIndex, bool) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	errFlag := true
	utxoIndex := lutxo.NewUTXOIndex(bc.GetUtxoCache())
	if !utxoIndex.UpdateUtxos(bc.GetTxPool().GetAllTransactions()) {
		logger.Warn("GetUpdatedUTXOIndex error")
		errFlag = false
	}
	return utxoIndex, errFlag
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
		logger.Error(err)
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

func (bc *Blockchain) GetBlockMutex() *sync.Mutex {
	return bc.mutex
}

func (bc *Blockchain) SetTailBlockHash(tailBlockHash hash.Hash) {
	bc.bc.SetTailBlockHash(tailBlockHash)
}

func (bc *Blockchain) SetState(state blockchain.BlockchainState) {
	bc.bc.SetState(state)
}

func (bc *Blockchain) GetState() blockchain.BlockchainState {
	return bc.bc.GetState()
}

func (bc *Blockchain) AddBlockContextToTail(ctx *BlockContext) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	tailBlockHash := bc.GetTailBlockHash()
	if ctx.Block.GetHeight() != 0 && bytes.Compare(ctx.Block.GetPrevHash(), tailBlockHash) != 0 {
		logger.WithFields(logger.Fields{
			"blockHeight": ctx.Block.GetHeight(),
		}).Warn("AddBlockContextToTail : prevhash verify failed.")
		return ErrPrevHashVerifyFailed
	}

	blockLogger := logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	})

	if err := bc.saveDataToDb(ctx); err != nil {
		return err
	}

	bc.updateLIB(ctx.Block.GetHeight())

	numTxBeforeExe := bc.GetTxPool().GetNumOfTxInPool()
	//Remove transactions in current transaction pool
	bc.GetTxPool().CleanUpMinedTxs(ctx.Block.GetTransactions())
	bc.GetTxPool().ResetPendingTransactions()

	logger.WithFields(logger.Fields{
		"num_txs_before_add_block":    numTxBeforeExe,
		"num_txs_after_update_txpool": bc.GetTxPool().GetNumOfTxInPool(),
	}).Info("Blockchain : update tx pool")

	blockLogger.WithFields(logger.Fields{
		"numOfTx": len(ctx.Block.GetTransactions()),
	}).Info("Blockchain: added a new block to tail.")

	return nil
}

func (bc *Blockchain) Iterator() *Blockchain {
	return &Blockchain{
		blockchain.NewBlockchain(bc.GetTailBlockHash(), bc.GetLIBHash()),
		bc.db,
		bc.utxoCache,
		bc.libPolicy,
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
		err = transaction.PutTxJournal(*tx, bc.db)
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

func (bc *Blockchain) IsFoundBeforeLib(hash hash.Hash) bool {
	bci := bc.Iterator()
	for {
		blk, err := bci.Next()
		if err != nil {
			return false
		}
		if blk.GetHash().Equals(hash) {
			return true
		}
		if blk.GetHash().Equals(bc.GetLIBHash()) {
			return false
		}
	}
}

//rollback the blockchain to a block with the targetHash
func (bc *Blockchain) Rollback(index *lutxo.UTXOIndex, targetHash hash.Hash, scState *scState.ScState) bool {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	parentblockHash := bc.GetTailBlockHash()
	//if is child of tail, skip rollback
	if lblock.IsHashEqual(parentblockHash, targetHash) {
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
			logger.Warn(err)
			return false
		}
		parentblockHash = block.GetPrevHash()

		for _, tx := range block.GetTransactions() {
			adaptedTx := transaction.NewTxAdapter(tx)
			if !adaptedTx.IsCoinbase() && !adaptedTx.IsRewardTx() && !adaptedTx.IsGasRewardTx() && !adaptedTx.IsGasChangeTx() {
				bc.txPool.Rollback(*tx)
			}
		}
	}

	err := bc.setTailBlockHash(parentblockHash)
	if err != nil {
		logger.Error("Blockchain: failed to set tail block hash during rollback!")
		return false
	}

	if err = scState.Save(parentblockHash); err != nil {
		logger.Warn(err)
		return false
	}

	//bc.db.EnableBatch()
	//updated utxo in db
	err = index.Save()
	if err != nil {
		logger.Warn(err)
		return false
	}
	bc.savedHash(UtxoSaveHash)
	//bc.db.DisableBatch()
	//bc.db.Flush()

	return true
}

func (bc *Blockchain) setTailBlockHash(hash hash.Hash) error {
	err := bc.db.Put(tailBlockHash, hash)
	if err != nil {
		return err
	}
	bc.bc.SetTailBlockHash(hash)
	return nil
}

func (bc *Blockchain) savedHash(bytes []byte) {
	err := bc.db.Put(bytes, bc.GetTailBlockHash())
	if err != nil {
		logger.Warn(err)
	}
}

func (bc *Blockchain) SetLIBHash(hash hash.Hash) {
	bc.bc.SetLIBHash(hash)
}

// GasPrice returns gas price in current blockchain
func (bc *Blockchain) GasPrice() uint64 {
	return DefaultGasPrice
}

func (bc *Blockchain) CheckMinProducerPolicy(blk *block.Block) bool {

	if bc.libPolicy.IsBypassingLibCheck() {
		return true
	}

	return bc.isAliveProducerSufficient(blk)

}

//isAliveProducerSufficient returns true if alive producers are greater than minimum producers(total *2/3)
func (bc *Blockchain) isAliveProducerSufficient(blk *block.Block) bool {
	minProduerNum := bc.libPolicy.GetMinConfirmationNum()
	onlineProducers := make(map[string]bool)
	currentCheckBlk := blk
	var err error
	if bc.GetMaxHeight() == 0 {
		return true
	}
	if bc.GetMaxHeight() < uint64(minProduerNum) {
		for i := uint64(0); i < bc.GetMaxHeight(); i++ {
			currentCheckBlk, err = bc.GetBlockByHash(currentCheckBlk.GetPrevHash())
			if err != nil {
				logger.WithError(err).Warn("Blockchain: Cant not read parent block while checking alive producer.")
				return false
			}
			if blk.GetProducer() == currentCheckBlk.GetProducer() {
				return false
			}
		}
	} else {
		onlineProducers[currentCheckBlk.GetProducer()] = true
		for i := 0; i < bc.libPolicy.GetTotalProducersNum()-1; i++ {
			currentCheckBlk, err = bc.GetBlockByHash(currentCheckBlk.GetPrevHash())
			if err != nil {
				logger.WithError(err).Warn("Blockchain: Cant not read parent block while checking alive producer")
				return false
			}
			if currentCheckBlk.GetHeight() == 0 {
				break
			}
			onlineProducers[currentCheckBlk.GetProducer()] = true
		}
		if len(onlineProducers) < minProduerNum {
			return false
		}
	}
	return true
}

func (bc *Blockchain) updateLIB(currBlkHeight uint64) {
	libHash, err := bc.getLIB(currBlkHeight)
	if err != nil {
		logger.Warn("updateLIB failed")
		return
	}
	bc.SetLIBHash(libHash)
}

func (bc *Blockchain) getLIB(currBlkHeight uint64) (hash.Hash, error) {
	if bc.libPolicy == nil {
		return []byte{}, errors.New("libPolicy is nil")
	}

	minConfirmationNum := bc.libPolicy.GetMinConfirmationNum()
	LIBHeight := uint64(0)
	if currBlkHeight > uint64(minConfirmationNum) {
		LIBHeight = currBlkHeight - uint64(minConfirmationNum)
	}

	LIBBlk, err := bc.GetBlockByHeight(LIBHeight)
	if err != nil {
		logger.WithError(err).Warn("Blockchain: Can not find LIB block in database")
		return []byte{}, err
	}

	return LIBBlk.GetHash(), nil
}

func (bc *Blockchain) DeleteBlockByHash(hash hash.Hash) {
	if err := bc.db.Del(hash); err != nil {
		logger.Warn("Delete the block failed.")
	}
}

func DataCheckingAndRecovery(db storage.Storage) error {
	logger.Info("Data checking ...")

	getHash := func(byte []byte) (hash.Hash, error) {
		hash, err := db.Get(byte)
		if err != nil {
			return hash, err
		}
		return hash, nil
	}

	tbHash, err := getHash(tailBlockHash)
	if err != nil {
		return err
	}
	uHash, err := getHash(UtxoSaveHash)
	if err != nil {
		return err
	}
	if !bytes.Equal(uHash, tbHash) { //checking and recovering utxo and scState
		logger.Info("Incomplete data found, recovering utxo and scState...")
		getBlock := func(hash hash.Hash) *block.Block {
			rawBytes, err := db.Get(hash)
			if err != nil {
				logger.Warn(err)
			}
			return block.Deserialize(rawBytes)
		}
		blk := getBlock(tbHash)
		parentBlk := getBlock(blk.GetPrevHash())
		contractStates := scState.NewScState(utxo.NewUTXOCache(db))
		utxo := lutxo.NewUTXOIndex(utxo.NewUTXOCache(db))
		if !lblock.VerifyTransactions(blk, utxo, contractStates, parentBlk, db) {
			logger.Warn("get check utxo failed")
		}
		if err = contractStates.Save(tbHash); err != nil { //recover scState
			logger.Warn(err)
		}

		utxo.SelfCheckingUTXO()
		err := utxo.Save()
		if err != nil {
			logger.Warn(err)
		}

		err = db.Put(UtxoSaveHash, tbHash)
		if err != nil {
			logger.Warn("put hash to db faild:", err)
		}

		logger.Info("Data have been recovered.")
		return nil
	}
	logger.Info("Data check completed.")
	return nil
}

func (bc *Blockchain) saveDataToDb(ctx *BlockContext) error {
	//Note that changing the code order will cause data checking and recovery errors
	err := bc.AddBlockToDb(ctx.Block) //1.save block
	if err != nil {
		logger.Warn("Failed to add block to db.")
		return err
	}

	err = bc.setTailBlockHash(ctx.Block.GetHash()) //2.save tail block hash
	if err != nil {
		logger.Error("Failed to set tail block hash!")
		return err
	}

	err = ctx.State.Save(ctx.Block.GetHash()) //3.save state and utxo
	if err != nil {
		logger.Warn("Failed to add scState to db.")
		return err
	}

	err = ctx.UtxoIndex.Save()
	if err != nil {
		logger.Warn("Failed to save utxo to db.")
		return err
	}
	bc.savedHash(UtxoSaveHash)

	return nil
}
