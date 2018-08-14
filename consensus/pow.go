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
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
	"github.com/dappley/go-dappley/network"
)

type ProofOfWork struct {
	bc 			*core.Blockchain
	miner 		*Miner
	mintBlkChan	chan(*MinedBlock)
	node    	*network.Node
	exitCh 		chan(bool)
}

func NewProofOfWork() *ProofOfWork{
	p := &ProofOfWork{
		miner:			NewMiner(),
		mintBlkChan: 	make(chan(*MinedBlock),1),
		node: 			nil,
		exitCh: 		make(chan(bool),1),
	}
	return p
}

func (pow *ProofOfWork) Setup(node *network.Node, cbAddr string){
	pow.bc = node.GetBlockchain()
	pow.node = node
	pow.miner.Setup(pow.bc, cbAddr, pow.mintBlkChan)
}

func (pow *ProofOfWork) GetNode() *network.Node{
	return pow.node
}

func (pow *ProofOfWork) SetTargetBit(bit int){
	pow.miner.SetTargetBit(bit)
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
			case minedBlk := <- pow.mintBlkChan:
				if minedBlk.isValid {
					pow.updateNewBlock(minedBlk.block)
					pow.bc.MergeFork()
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

func (pow *ProofOfWork) Validate(blk *core.Block) bool {
	return pow.miner.Validate(blk)
}

func (pow *ProofOfWork) updateNewBlock(newBlock *core.Block){
	logger.Info("PoW: Minted a new block. height:", newBlock.GetHeight())
	if !newBlock.VerifyHash(){
		logger.Warn("hash verification is wrong")

	}
	pow.bc.UpdateNewBlock(newBlock)
	pow.broadcastNewBlock(newBlock)
}

func (pow *ProofOfWork) broadcastNewBlock(blk *core.Block){
	//broadcast the block to other nodes
	pow.node.SendBlock(blk)
}

func (pow *ProofOfWork) StartNewBlockMinting(){
	pow.miner.Stop()
}