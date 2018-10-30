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

	"github.com/dappley/go-dappley/core"
)

type Delegate struct {
	exitCh      chan bool
	bc          *core.Blockchain
	beneficiary string
	key         string
	newBlock    *NewBlock
	requirement Requirement
	newBlockCh  chan *NewBlock
	idle        bool
}

func NewDelegate() *Delegate {
	return &Delegate{
		exitCh:      make(chan bool, 1),
		bc:          nil,
		beneficiary: "",
		newBlock:    &NewBlock{nil, false},
		requirement: noRequirement,
		idle:        true,
	}
}

func (d *Delegate) SetPrivateKey(key string) {
	d.key = key
}

func (d *Delegate) Beneficiary() string {
	return d.beneficiary
}

func (d *Delegate) SetRequirement(requirement Requirement) {
	d.requirement = requirement
}

func (d *Delegate) Setup(bc *core.Blockchain, beneficiaryAddr string, newBlockCh chan *NewBlock) {
	d.bc = bc
	d.beneficiary = beneficiaryAddr
	d.newBlockCh = newBlockCh
}

func (d *Delegate) Start() {
	go func() {
		if d.bc.GetBlockPool().GetSyncState() {
			return
		}
		logger.Info("Delegate: Producing a block...")
		d.resetExitCh()
		d.idle = false
		d.prepare()
		select {
		case <-d.exitCh:
			logger.Warn("Delegate: Block production is interrupted")
		default:
			d.produceBlock()
		}
		d.returnBlk()
		d.idle = true
		logger.Info("Delegate: Produced a block")
	}()
}

func (d *Delegate) Stop() {
	if len(d.exitCh) == 0 {
		d.exitCh <- true
	}
}

func (d *Delegate) IsIdle() bool {
	return d.idle
}

func (d *Delegate) prepare() {
	d.newBlock = d.prepareBlock()
}

func (d *Delegate) returnBlk() {
	if !d.newBlock.IsValid {
		d.newBlock.Rollback(d.bc.GetTxPool())
	}
	d.newBlockCh <- d.newBlock
}

func (d *Delegate) resetExitCh() {
	if len(d.exitCh) > 0 {
		<-d.exitCh
	}
}

func (d *Delegate) prepareBlock() *NewBlock {

	parentBlock, err := d.bc.GetTailBlock()
	if err != nil {
		logger.Error(err)
	}

	//verify all transactions
	d.verifyTransactions()
	//get all transactions
	txs := d.bc.GetTxPool().Pop()
	//add coinbase transaction to transaction pool
	cbtx := core.NewCoinbaseTX(d.beneficiary, "", d.bc.GetMaxHeight()+1)
	txs = append(txs, &cbtx)
	// TODO: add tips to txs

	//prepare the new block
	return &NewBlock{core.NewBlock(txs, parentBlock), false}
}

// produceBlock hashes and signs the new block; returns true if it was successful and the block fulfills the requirement
func (d *Delegate) produceBlock() bool {
	hash := d.newBlock.CalculateHashWithoutNonce()
	d.newBlock.SetHash(hash)
	d.newBlock.SetNonce(0)
	fulfilled := d.requirement(d.newBlock.Block)
	if fulfilled {
		if len(d.key) > 0 {
			signed := d.newBlock.SignBlock(d.key, hash)
			if !signed {
				logger.Warn("Delegate Key= ", d.key)
				return false
			}
		}
		d.newBlock.IsValid = true
	}
	return fulfilled
}

// verifyTransactions removes invalid transactions from transaction pool
func (d *Delegate) verifyTransactions() {
	utxoPool := core.LoadUTXOIndex(d.bc.GetDb())
	txPool := d.bc.GetTxPool()
	txPool.RemoveInvalidTransactions(utxoPool)
}
