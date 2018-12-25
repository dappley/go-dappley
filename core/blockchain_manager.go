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
	peer "github.com/libp2p/go-libp2p-peer"
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
	if !(bm.blockchain.GetConsensus().Validate(block)) {
		logger.Warn("BlockPool: The received block is invalid according to consensus!")
		return false
	}
	logger.Debug("BlockPool: Block has been verified")
	return true
}
func (bm *BlockChainManager) Push(block *Block, pid peer.ID) {
	if bm.blockchain.GetState() != BlockchainReady {
		logger.Info("Blockchain not ready, discard received block")
		return
	}

	if !bm.VerifyBlock(block) {
		return
	}

	tree, _ := common.NewTree(block.GetHash().String(), block)
	logger.WithFields(logger.Fields{
		"From": pid.String(),
		"hash": hex.EncodeToString(block.GetHash()),
	}).Info("BlockPool: Received a new block: ")
	forkheadParentHash := bm.blockPool.CacheBlock(tree, bm.blockchain.GetMaxHeight())
	if forkheadParentHash == nil {
		return
	}
	if parent, _ := bm.blockchain.GetBlockByHash(forkheadParentHash); parent == nil {
		bm.blockPool.requestPrevBlock(tree, pid)
		return
	}
	forkBlks := bm.blockPool.GenerateForkBlocks(tree, bm.blockchain.GetMaxHeight())
	bm.blockchain.SetState(BlockchainSync)
	bm.MergeFork(forkBlks, forkheadParentHash)
	bm.blockPool.CleanCache(tree)
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
		logger.Error("Corrupt blockchain, please delete DB file and resynchronize to the network")
		return err
	}
	parentBlk, err := bm.blockchain.GetBlockByHash(forkParentHash)
	if !bm.VerifyTransactions(*utxo, scState, forkBlks, parentBlk) {
		logger.Error("MergeFork failed, transaction verify failed.")
		return ErrTransactionVerifyFailed
	}

	bm.blockchain.Rollback(forkParentHash)

	//add all blocks in fork from head to tail
	bm.blockchain.addBlocksToTail(forkBlks)

	return nil
}

//Verify all transactions in a fork
func (bm *BlockChainManager) VerifyTransactions(utxoSnapshot UTXOIndex, scState *ScState, forkBlks []*Block, parentBlk *Block) bool {
	logger.Info("Verifying transactions")
	for i := len(forkBlks) - 1; i >= 0; i-- {
		logger.WithFields(logger.Fields{
			"height": forkBlks[i].GetHeight(),
			"hash":   hex.EncodeToString(forkBlks[i].GetHash()),
		}).Debug("Verifying block before merge")

		if !forkBlks[i].VerifyTransactions(utxoSnapshot, scState, bm.blockchain.GetSCManager(), parentBlk) {
			return false
		}
		parentBlk = forkBlks[i]
		utxoSnapshot.UpdateUtxoState(forkBlks[i].GetTransactions())
	}
	return true
}
