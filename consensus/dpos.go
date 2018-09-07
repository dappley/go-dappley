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
	"time"
	logger "github.com/sirupsen/logrus"
)

type Dpos struct{
	bc        *core.Blockchain
	miner     *Miner
	mintBlkCh chan(*MinedBlock)
	node      core.NetService
	quitCh    chan(bool)
	dynasty   *Dynasty
}

func NewDpos() *Dpos{
	dpos := &Dpos{
		miner:     NewMiner(),
		mintBlkCh: make(chan(*MinedBlock),1),
		node:      nil,
		quitCh:    make(chan(bool),1),
	}
	return dpos
}

func (dpos *Dpos) Setup(node core.NetService, cbAddr string){
	dpos.bc = node.GetBlockchain()
	dpos.node = node
	dpos.miner.Setup(dpos.bc, cbAddr, dpos.mintBlkCh)
}

func (dpos *Dpos) SetTargetBit(bit int){
	dpos.miner.SetTargetBit(bit)
}

func (dpos *Dpos) SetDynasty(dynasty *Dynasty){
	dpos.dynasty = dynasty
}

func (dpos *Dpos) GetDynasty() *Dynasty{
	return dpos.dynasty
}

func (dpos *Dpos) Validate(block *core.Block) bool{
	return dpos.miner.Validate(block) && dpos.dynasty.ValidateProducer(block)
}

func (dpos *Dpos) Start(){
	go func(){
		logger.Info("Dpos Starts...")
		ticker := time.NewTicker(time.Second).C
		for{
			select{
			case now := <- ticker:
				if dpos.dynasty.IsMyTurn(dpos.miner.cbAddr, now.Unix()){
					logger.Info("Dpos: My Turn to Mint!")
					dpos.miner.Start()
				}
			case minedBlk := <- dpos.mintBlkCh:
				if minedBlk.isValid {
					logger.Info("Dpos: A Block has been mined!")
					dpos.updateNewBlock(minedBlk.block)
					dpos.bc.MergeFork()
				}
			case <-dpos.quitCh:
				logger.Info("Dpos: Dpos Stops!")
				return
			}
		}
	}()
}

func (dpos *Dpos) Stop() {
	dpos.quitCh <- true
	dpos.miner.Stop()
}

func (dpos *Dpos) StartNewBlockMinting(){
	dpos.miner.Stop()
}

func (dpos *Dpos) updateNewBlock(newBlock *core.Block){
	logger.Info("DPoS: Minted a new block. height:", newBlock.GetHeight())
	dpos.bc.AddBlockToTail(newBlock)
	dpos.node.SendBlock(newBlock)
}

