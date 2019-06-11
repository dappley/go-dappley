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

	peer "github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
)

const HeightDiffThreshold = 10

type BlockChainManager struct {
	blockchain *Blockchain
	blockPool  *BlockPool
}

func NewBlockChainManager() *BlockChainManager {
	return &BlockChainManager{}
}

func (bm *BlockChainManager) SetblockPool(blockPool *BlockPool) {
	bm.blockPool = blockPool
}

func (bm *BlockChainManager) Setblockchain(blockchain *Blockchain) {
	bm.blockchain = blockchain
}

func (bm *BlockChainManager) Getblockchain() *Blockchain {
	return bm.blockchain
}

func (bm *BlockChainManager) GetblockPool() *BlockPool {
	return bm.blockPool
}

func (bm *BlockChainManager) VerifyBlock(block *Block) bool {
	if !bm.blockPool.Verify(block) {
		return false
	}
	logger.Debug("BlockChainManager: block is verified.")
	if !(bm.blockchain.GetConsensus().Validate(block)) {
		logger.Warn("BlockChainManager: block is invalid according to consensus!")
		return false
	}
	logger.Debug("BlockChainManager: block is valid according to consensus.")
	return true
}

func (bm *BlockChainManager) Push(block *Block, pid peer.ID) {
	logger.WithFields(logger.Fields{
		"from":   pid.String(),
		"hash":   hex.EncodeToString(block.GetHash()),
		"height": block.GetHeight(),
	}).Info("BlockChainManager: received a new block.")

	if bm.blockchain.GetState() != BlockchainReady {
		logger.Info("BlockChainManager: Blockchain not ready, discard received block")
		return
	}
	if !bm.VerifyBlock(block) {
		return
	}

	receiveBlockHeight := block.GetHeight()
	ownBlockHeight := bm.Getblockchain().GetMaxHeight()
	if receiveBlockHeight >= ownBlockHeight &&
		receiveBlockHeight-ownBlockHeight >= HeightDiffThreshold &&
		bm.blockchain.GetState() == BlockchainReady {
		logger.Info("The height of the received block is higher than the height of its own block,to start download blockchain")
		bm.blockPool.DownloadBlocksCh() <- true
		return
	}

	tree, _ := common.NewTree(block.GetHash().String(), block)
	forkHead := bm.blockPool.CacheBlock(tree, bm.blockchain.GetMaxHeight())
	forkHeadParentHash := forkHead.GetValue().(*Block).GetPrevHash()
	if forkHeadParentHash == nil {
		return
	}
	parent, _ := bm.blockchain.GetBlockByHash(forkHeadParentHash)
	if parent == nil {
		logger.WithFields(logger.Fields{
			"parent_hash":   forkHeadParentHash,
			"parent_height": forkHead.GetValue().(*Block).GetHeight() - 1,
		}).Info("BlockChainManager: cannot find the parent of the received block from blockchain.")
		bm.blockPool.requestPrevBlock(forkHead, pid)
		return
	}
	forkBlks := bm.blockPool.GenerateForkBlocks(forkHead, bm.blockchain.GetMaxHeight())
	bm.blockchain.SetState(BlockchainSync)
	_ = bm.MergeFork(forkBlks, forkHeadParentHash)
	bm.blockPool.CleanCache(forkHead)
	bm.blockchain.SetState(BlockchainReady)
}

func (bm *BlockChainManager) MergeFork(forkBlks []*Block, forkParentHash Hash) error {

	//find parent block
	if len(forkBlks) == 0 {
		return nil
	}
	forkHeadBlock := forkBlks[len(forkBlks)-1]
	if forkHeadBlock == nil {
		return nil
	}

	//verify transactions in the fork
	utxo, scState, err := RevertUtxoAndScStateAtBlockHash(bm.blockchain.db, bm.blockchain, forkParentHash)
	if err != nil {
		logger.Error("BlockChainManager: blockchain is corrupted! Delete the database file and resynchronize to the network.")
		return err
	}
	rollBackUtxo := utxo.DeepCopy()
	rollScState := scState.DeepCopy()

	parentBlk, err := bm.blockchain.GetBlockByHash(forkParentHash)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
			"hash":  hex.EncodeToString(forkParentHash),
		}).Error("BlockChainManager: get fork parent block failed.")
	}

	firstCheck := true

	for i := len(forkBlks) - 1; i >= 0; i-- {
		logger.WithFields(logger.Fields{
			"height": forkBlks[i].GetHeight(),
			"hash":   hex.EncodeToString(forkBlks[i].GetHash()),
		}).Debug("BlockChainManager: is verifying a block in the fork.")

		if !forkBlks[i].VerifyTransactions(utxo, scState, bm.blockchain.GetSCManager(), parentBlk) {
			return ErrTransactionVerifyFailed
		}

		if firstCheck {
			firstCheck = false
			bm.blockchain.Rollback(forkParentHash, rollBackUtxo, rollScState)
		}

		ctx := BlockContext{Block: forkBlks[i], UtxoIndex: utxo, State: scState}
		err = bm.blockchain.AddBlockContextToTail(&ctx)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error":  err,
				"height": forkBlks[i].GetHeight(),
			}).Error("BlockChainManager: add fork to tail failed.")
		}
		parentBlk = forkBlks[i]
	}

	return nil
}

// RevertUtxoAndScStateAtBlockHash returns the previous snapshot of UTXOIndex when the block of given hash was the tail block.
func RevertUtxoAndScStateAtBlockHash(db storage.Storage, bc *Blockchain, hash Hash) (*UTXOIndex, *ScState, error) {
	index := NewUTXOIndex(bc.GetUtxoCache())
	scState := LoadScStateFromDatabase(db)
	bci := bc.Iterator()

	// Start from the tail of blockchain, compute the previous UTXOIndex by undoing transactions
	// in the block, until the block hash matches.
	for {
		block, err := bci.Next()

		if bytes.Compare(block.GetHash(), hash) == 0 {
			break
		}

		if err != nil {
			return nil, nil, err
		}

		if len(block.GetPrevHash()) == 0 {
			return nil, nil, ErrBlockDoesNotExist
		}

		err = index.UndoTxsInBlock(block, bc, db)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"hash": block.GetHash(),
			}).Warn("BlockChainManager: failed to calculate previous state of UTXO index for the block")
			return nil, nil, err
		}

		err = scState.RevertState(db, block.GetHash())
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"hash": block.GetHash(),
			}).Warn("BlockChainManager: failed to calculate previous state of scState for the block")
			return nil, nil, err
		}
	}

	return index, scState, nil
}

/* NumForks returns the number of forks in the BlockPool and the height of the current longest fork */
func (bm *BlockChainManager) NumForks() (int64, int64) {
	var numForks, maxHeight int64 = 0, 0

	bm.blockPool.ForkHeadRange(func(blkHash string, tree *common.Tree) {
		rootBlk := tree.GetValue().(*Block)
		_, err := bm.blockchain.GetBlockByHash(rootBlk.GetPrevHash())
		if err == nil {
			/* the cached block is rooted in the BlockChain */
			numForks += tree.NumLeaves()
			t := tree.Height()
			if t > maxHeight {
				maxHeight = t
			}
		}
	})

	return numForks, maxHeight
}
