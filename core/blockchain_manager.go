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
	"encoding/hex"

	"github.com/dappley/go-dappley/common"
	"github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"
)

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
	tree, _ := common.NewTree(block.GetHash().String(), block)
	bm.blockPool.CacheBlock(tree, bm.blockchain.GetMaxHeight())
	forkHead := tree.GetRoot()
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
	scState := NewScState()
	scState.LoadFromDatabase(bm.blockchain.db, forkParentHash)

	//verify transactions in the fork
	utxo, err := GetUTXOIndexAtBlockHash(bm.blockchain.db, bm.blockchain, forkParentHash)
	if err != nil {
		logger.Error("BlockChainManager: blockchain is corrupted! Delete the database file and resynchronize to the network.")
		return err
	}
	rollBackUtxo := utxo.DeepCopy()

	parentBlk, err := bm.blockchain.GetBlockByHash(forkParentHash)
	if !bm.VerifyTransactions(utxo, scState, forkBlks, parentBlk) {
		logger.Errorf("BlockChainManager: Verify fork blocks transaction failed with parent height %v", parentBlk.GetHeight())
		return ErrTransactionVerifyFailed
	}

	bm.blockchain.Rollback(forkParentHash, rollBackUtxo)

	//add all blocks in fork from head to tail
	bm.blockchain.addBlocksToTail(forkBlks)
	return nil
}

//Verify all transactions in a fork
func (bm *BlockChainManager) VerifyTransactions(utxoIndex *UTXOIndex, scState *ScState, forkBlks []*Block, parentBlk *Block) bool {
	logger.Info("BlockChainManager: is verifying transactions...")
	for i := len(forkBlks) - 1; i >= 0; i-- {
		logger.WithFields(logger.Fields{
			"height": forkBlks[i].GetHeight(),
			"hash":   hex.EncodeToString(forkBlks[i].GetHash()),
		}).Debug("BlockChainManager: is verifying a block in the fork.")

		if !forkBlks[i].VerifyTransactions(utxoIndex, scState, parentBlk, bm.blockchain.GetTxPool()) {
			return false
		}
		parentBlk = forkBlks[i]
		utxoIndex.UpdateUtxoState(forkBlks[i].GetTransactions())
	}
	return true
}
