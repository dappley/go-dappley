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
	"bytes"
	"encoding/hex"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
)

const version = byte(0x00)

type DPOS struct {
	bc         *core.Blockchain
	delegate   BlockProducer
	newBlockCh chan *NewBlock
	node       core.NetService
	quitCh     chan bool
	dynasty    *Dynasty
	slot       *lru.Cache
}

func NewDPOS() *DPOS {
	dpos := &DPOS{
		delegate:   NewDelegate(),
		newBlockCh: make(chan *NewBlock, 1),
		node:       nil,
		quitCh:     make(chan bool, 1),
	}

	slot, err := lru.New(128)
	if err != nil {
		logger.Panic(err)
	}
	dpos.slot = slot
	return dpos
}

func (dpos *DPOS) GetSlot() *lru.Cache {
	return dpos.slot
}

func (dpos *DPOS) Setup(node core.NetService, cbAddr string) {
	dpos.bc = node.GetBlockchain()
	dpos.node = node
	dpos.delegate.Setup(dpos.bc, cbAddr, dpos.newBlockCh)
	dpos.delegate.SetRequirement(dpos.requirementForNewBlock)
}

func (dpos *DPOS) SetKey(key string) {
	dpos.delegate.SetPrivateKey(key)
}

func (dpos *DPOS) SetDynasty(dynasty *Dynasty) {
	dpos.dynasty = dynasty
}

func (dpos *DPOS) GetDynasty() *Dynasty {
	return dpos.dynasty
}

func (dpos *DPOS) AddProducer(producer string) error {
	err := dpos.dynasty.AddProducer(producer)
	return err
}

func (dpos *DPOS) GetProducers() []string {
	return dpos.dynasty.GetProducers()
}

func (dpos *DPOS) GetBlockChain() *core.Blockchain {
	return dpos.bc
}

func (dpos *DPOS) requirementForNewBlock(block *core.Block) bool {
	if !dpos.beneficiaryIsProducer(block) {
		logger.Debug("DPoS: producer validate failed")
		return false
	}
	if dpos.isDoubleMint(block) {
		logger.Debug("DPoS: doubleminting case found!")
		return false
	}

	return true
}

// Validate checks that the block fulfills the dpos requirement and accepts the block in the time slot
func (dpos *DPOS) Validate(block *core.Block) bool {
	fulfilled := dpos.requirementForNewBlock(block)
	if !fulfilled {
		return false
	}

	dpos.slot.Add(block.GetTimestamp(), block)
	return true
}

func (dpos *DPOS) Start() {
	go func() {
		logger.Info("DPoS Starts...", dpos.node.GetPeerID())
		ticker := time.NewTicker(time.Second).C
		for {
			select {
			case now := <-ticker:
				if dpos.dynasty.IsMyTurn(dpos.delegate.Beneficiary(), now.Unix()) {
					logger.WithFields(logger.Fields{
						"peerid": dpos.node.GetPeerID(),
					}).Info("My Turn to Mint")
					dpos.delegate.Start()
				}
			case newBlk := <-dpos.newBlockCh:
				if newBlk.IsValid {
					logger.WithFields(logger.Fields{
						"peerid": dpos.node.GetPeerID(),
						"hash":   hex.EncodeToString(newBlk.GetHash()),
					}).Info("DPoS: A Block is produced!")
					dpos.updateNewBlock(newBlk.Block)
				}
			case <-dpos.quitCh:
				logger.WithFields(logger.Fields{
					"peerid": dpos.node.GetPeerID(),
				}).Info("DPoS: DPoS Stops!")
				return
			}
		}
	}()
}

func (dpos *DPOS) Stop() {
	dpos.quitCh <- true
	dpos.delegate.Stop()
}

func (dpos *DPOS) isForking() bool {
	return false
}

func (dpos *DPOS) isDoubleMint(block *core.Block) bool {
	if _, exist := dpos.slot.Get(block.GetTimestamp()); exist {
		logger.Debug("Someone is minting when they are not supposed to!")
		return true
	}
	return false
}

// verifyProducer verifies a given block is produced by the valid producer by verifying the signature of the block
func (dpos *DPOS) verifyProducer(block *core.Block) bool {
	if block == nil {
		logger.Warn("DPoS: block is empty!")
		return false
	}

	hash := block.GetHash()
	sign := block.GetSign()

	producer := dpos.dynasty.ProducerAtATime(block.GetTimestamp())

	if hash == nil {
		logger.Warn("DPoS: block hash empty!")
		return false
	}
	if sign == nil {
		logger.Warn("DPoS: block signature empty!")
		return false
	}

	pubkey, err := secp256k1.RecoverECDSAPublicKey(hash, sign)
	if err != nil {
		logger.Warn("DPoS: Get pub key from block signature error!")
		return false
	}

	pubKeyHash,err := core.NewUserPubKeyHash(pubkey[1:])
	if err != nil {
		logger.Warn("DPoS: Invalid Public Key!")
		return false
	}

	address := pubKeyHash.GenerateAddress()

	if strings.Compare(address.String(), producer) != 0 {
		logger.Warn("DPoS: Address is not current producer's")
		return false
	}

	return true
}

// beneficiaryIsProducer is a Requirement that ensures the reward is paid to the producer at the time slot
func (dpos *DPOS) beneficiaryIsProducer(block *core.Block) bool {
	if block == nil {
		logger.Debug("beneficiaryIsProducer requirement failed: block is empty")
		return false
	}

	producer := dpos.dynasty.ProducerAtATime(block.GetTimestamp())
	producerHash := core.HashAddress(producer)

	cbtx := block.GetCoinbaseTransaction()
	if cbtx == nil {
		logger.Debug("beneficiaryIsProducer requirement failed: coinbase tx is empty")
		return false
	}

	if len(cbtx.Vout) == 0 {
		logger.Debug("beneficiaryIsProducer requirement failed: coinbase Vout is empty")
		return false
	}

	return bytes.Compare(producerHash, cbtx.Vout[0].PubKeyHash.GetPubKeyHash()) == 0
}

func (dpos *DPOS) IsProducingBlock() bool {
	return !dpos.delegate.IsIdle()
}

func (dpos *DPOS) updateNewBlock(newBlock *core.Block) {
	logger.WithFields(logger.Fields{
		"height": newBlock.GetHeight(),
		"hash":   hex.EncodeToString(newBlock.GetHash()),
	}).Info("DPoS: Minted a new block")
	dpos.bc.AddBlockToTail(newBlock)
	dpos.node.BroadcastBlock(newBlock)
}

func (dpos *DPOS) VerifyBlock(block *core.Block) bool {
	return dpos.verifyProducer(block)
}
