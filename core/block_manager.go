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

type BlockManager struct {
	blockchain *Blockchain
	blockPool  *BlockPool
}

func NewBlockManager() *BlockManager {
	return &BlockManager{}
}

func (bm *BlockManager) SetblockPool(blockPool *BlockPool) {
	bm.blockPool = blockPool
}

func (bm *BlockManager) Setblockchain(blockchain *Blockchain) {
	bm.blockchain = blockchain
}

func (bm *BlockManager) Getblockchain() *Blockchain {
	return bm.blockchain
}

func (bm *BlockManager) GetblockPool() *BlockPool {
	return bm.blockPool
}

func (bm *BlockManager) Push(block *Block, pid peer.ID) {
	if !bm.blockPool.Verify(block) {
		return
	}
	if !(bm.blockchain.GetConsensus().Validate(block)) {
		logger.Warn("BlockPool: The received block is invalid according to consensus!")
		return
	}
	logger.Debug("BlockPool: Block has been verified")
	tree, _ := common.NewTree(block.GetHash().String(), block)
	logger.WithFields(logger.Fields{
		"From": pid.String(),
		"hash": hex.EncodeToString(block.GetHash()),
	}).Info("BlockPool: Received a new block: ")
	forkheadParentHash := bm.blockPool.HandleRecvdBlock(tree, bm.blockchain.GetMaxHeight())
	if forkheadParentHash == nil {
		return
	}
	if parent, _ := bm.blockchain.GetBlockByHash(forkheadParentHash); parent == nil {
		bm.blockPool.requestPrevBlock(tree, pid)
		return
	}
	forkBlks := bm.blockPool.GenerateForkBlocks(tree, bm.blockchain.GetMaxHeight())
	bm.blockchain.MergeFork(forkBlks, forkheadParentHash)
	bm.blockPool.CleanCache(tree)
}
