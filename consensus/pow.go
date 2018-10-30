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

package consensus

import (
	logger "github.com/sirupsen/logrus"
	"math/big"

	"github.com/dappley/go-dappley/core"
)

const defaultTargetBits = 0

type ProofOfWork struct {
	bc          *core.Blockchain
	miner       BlockProducer
	mintBlkChan chan *MinedBlock
	target      *big.Int
	blkProduced bool
	node        core.NetService
	exitCh      chan bool
}

func NewProofOfWork() *ProofOfWork {
	p := &ProofOfWork{
		miner:       NewMiner(),
		mintBlkChan: make(chan *MinedBlock, 1),
		blkProduced: false,
		node:        nil,
		exitCh:      make(chan bool, 1),
	}
	p.SetKey("")
	return p
}

func (pow *ProofOfWork) Setup(node core.NetService, cbAddr string) {
	pow.bc = node.GetBlockchain()
	pow.node = node
	pow.miner.Setup(pow.bc, cbAddr, pow.mintBlkChan)
}

func (pow *ProofOfWork) SetTargetBit(bit int) {
	if bit < 0 || bit > 256 {
		return
	}
	target := big.NewInt(1)
	pow.target = target.Lsh(target, uint(256-bit))

	pow.miner.SetRequirement(pow.isHashBelowTarget)
}

func (pow *ProofOfWork) SetKey(key string) {
	pow.miner.SetPrivKey(key)
}

func (pow *ProofOfWork) Start() {
	go func() {
		logger.Info("PoW started...")
		pow.miner.Start()
		for {
			select {
			case <-pow.exitCh:
				logger.Info("PoW stopped...")
				return
			case minedBlk := <-pow.mintBlkChan:
				pow.blkProduced = true
				if minedBlk.isValid {
					pow.updateNewBlock(minedBlk.block)
				}
				pow.miner.Start()
			}
		}
	}()
}

func (pow *ProofOfWork) Stop() {
	pow.exitCh <- true
	pow.miner.Stop()
}
func (pow *ProofOfWork) FinishedMining() bool {
	return pow.blkProduced
}

func (pow *ProofOfWork) isHashBelowTarget(block *core.Block) bool {
	var hashInt big.Int

	hash := block.GetHash()
	hashInt.SetBytes(hash)

	return hashInt.Cmp(pow.target) == -1
}

func (pow *ProofOfWork) Validate(block *core.Block) bool {
	return pow.isHashBelowTarget(block)
}

func (pow *ProofOfWork) updateNewBlock(newBlock *core.Block) {
	logger.Info("PoW: Minted a new block. height:", newBlock.GetHeight())
	if !newBlock.VerifyHash() {
		logger.Warn("hash verification is wrong")

	}
	pow.bc.AddBlockToTail(newBlock)
	pow.broadcastNewBlock(newBlock)
}

func (pow *ProofOfWork) broadcastNewBlock(blk *core.Block) {
	//broadcast the block to other nodes
	pow.node.BroadcastBlock(blk)
}

func (pow *ProofOfWork) StartNewBlockMinting() {
	pow.miner.Stop()
}

func (pow *ProofOfWork) VerifyBlock(block *core.Block) bool {
	return true
}

func (pow *ProofOfWork) AddProducer(producer string) error {
	return nil
}

func (pow *ProofOfWork) GetProducers() []string {
	return nil
}
