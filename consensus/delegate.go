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
	"math/big"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/core"
)

type Delegate struct {
	exitCh      chan bool
	bc          *core.Blockchain
	beneficiary string
	key         string
	newBlock    *MinedBlock
	retChan     chan *MinedBlock
}

func NewDelegate() *Delegate {
	return &Delegate{
		exitCh:      make(chan bool, 1),
		bc:          nil,
		beneficiary: "",
		newBlock:    &MinedBlock{nil, false},
	}
}

func (d *Delegate) SetPrivKey(key string) {
	d.key = key
}

func (d *Delegate) GetPrivKey() string {
	return d.key
}

func (d *Delegate) Beneficiary() string {
	return d.beneficiary
}

func (d *Delegate) Setup(bc *core.Blockchain, beneficiaryAddr string, retChan chan *MinedBlock) {
	d.bc = bc
	d.beneficiary = beneficiaryAddr
	d.retChan = retChan
}

func (d *Delegate) Start() {
	go func() {
		if d.bc.GetBlockPool().GetSyncState() {
			return
		}
		logger.Info("Delegate: Producing a block...")
		d.resetExitCh()
		d.prepare()
		select {
		case <-d.exitCh:
			logger.Warn("Delegate: Block production is interrupted")
		default:
			d.produceBlock()
		}
		d.returnBlk()
		logger.Info("Delegate: Produced a block")
	}()
}

func (d *Delegate) Stop() {
	if len(d.exitCh) == 0 {
		d.exitCh <- true
	}
}

func (d *Delegate) Validate(blk *core.Block) bool {
	var hashInt big.Int

	hash := blk.GetHash()
	hashInt.SetBytes(hash)

	//isValid := hashInt.Cmp(d.target) == -1

	return true
}

func (d *Delegate) prepare() {
	d.newBlock = d.prepareBlock()
}

func (d *Delegate) returnBlk() {
	if !d.newBlock.isValid {
		d.newBlock.block.Rollback(d.bc.GetTxPool())
	}
	d.retChan <- d.newBlock
}

func (d *Delegate) resetExitCh() {
	if len(d.exitCh) > 0 {
		<-d.exitCh
	}
}

func (d *Delegate) prepareBlock() *MinedBlock {

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
	return &MinedBlock{core.NewBlock(txs, parentBlock), false}
}

// produceBlock hashes and signs the new block; returns true if it was successful
func (d *Delegate) produceBlock() bool {
	hash := d.newBlock.block.CalculateHashWithoutNonce()
	d.newBlock.block.SetHash(hash)
	d.newBlock.block.SetNonce(0)
	keyString := d.GetPrivKey()
	if len(keyString) > 0 {
		signed := d.newBlock.block.SignBlock(keyString, hash)
		if !signed {
			logger.Warn("Miner Key= ", keyString)
			return false
		}
	}
	d.newBlock.isValid = true
	return true
}

// verifyTransactions removes invalid transactions from transaction pool
func (d *Delegate) verifyTransactions() {
	utxoPool := core.LoadUTXOIndex(d.bc.GetDb())
	txPool := d.bc.GetTxPool()
	txPool.RemoveInvalidTransactions(utxoPool)
}
