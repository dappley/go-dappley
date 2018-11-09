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
	"github.com/dappley/go-dappley/contract"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
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

	//return valid transactions
	utxoPool := core.LoadUTXOIndex(d.bc.GetDb())
	utxoTemp := utxoPool.DeepCopy()
	validTxs := d.bc.GetTxPool().ValidTxns(func(tx core.Transaction) bool {
		return tx.Verify(&utxoTemp, 0) && utxoTemp.ApplyTransaction(&tx) == nil
	})

	cbtx := d.calculateTips(validTxs)
	d.executeSmartContract(validTxs)
	validTxs = append(validTxs, cbtx)

	//prepare the new block
	return &NewBlock{core.NewBlock(validTxs, parentBlock), false}
}

func (d *Delegate) calculateTips(txs []*core.Transaction) *core.Transaction{
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(common.NewAmount(tx.Tip))
	}
	cbtx := core.NewCoinbaseTX(d.beneficiary, "", d.bc.GetMaxHeight()+1, totalTips)
	return &cbtx
}

//executeSmartContract executes all smart contracts
func (d *Delegate) executeSmartContract(txs []*core.Transaction){
	//start a new smart contract engine
	utxoIndex := core.LoadUTXOIndex(d.bc.GetDb())
	scStorage := core.NewScState()
	scStorage.LoadFromDatabase(d.bc.GetDb())
	engine := sc.NewV8Engine()
	for _, tx := range txs {
		tx.Execute(utxoIndex, scStorage, engine)
	}
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
