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

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"
)

type DPOS struct {
	bc          *core.Blockchain
	bp          *BlockProducer
	producerKey string
	newBlockCh  chan *core.Block
	node        core.NetService
	stopCh      chan bool
	dynasty     *Dynasty
	slot        *lru.Cache
}

func NewDPOS() *DPOS {
	dpos := &DPOS{
		bp:         NewBlockProducer(),
		newBlockCh: make(chan *core.Block, 1),
		node:       nil,
		stopCh:     make(chan bool, 1),
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
	dpos.bp.Setup(dpos.bc, cbAddr)
	dpos.bp.SetProcess(dpos.hashAndSign)
}

func (dpos *DPOS) SetKey(key string) {
	dpos.producerKey = key
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

// Validate checks that the block fulfills the dpos requirement and accepts the block in the time slot
func (dpos *DPOS) Validate(block *core.Block) bool {
	producerIsValid := dpos.verifyProducer(block)
	if !producerIsValid {
		return false
	}

	if !dpos.beneficiaryIsProducer(block) {
		logger.Debug("DPoS: Producer validation failed")
		return false
	}
	if dpos.isDoubleMint(block) {
		logger.Debug("DPoS: Double-minting case found!")
		return false
	}

	dpos.slot.Add(block.GetTimestamp(), block)
	return true
}

func (dpos *DPOS) Start() {
	go func() {
		logger.Info("DPoS starts...", dpos.node.GetPeerID())
		if len(dpos.stopCh) > 0 {
			<-dpos.stopCh
		}
		ticker := time.NewTicker(time.Second).C
		for {
			select {
			case now := <-ticker:
				if dpos.dynasty.IsMyTurn(dpos.bp.Beneficiary(), now.Unix()) {
					logger.WithFields(logger.Fields{
						"peerid": dpos.node.GetPeerID(),
					}).Info("DPoS: My Turn to produce block...")
					// Do not produce block if block pool is syncing
					if dpos.bp.bc.GetBlockPool().GetSyncState() {
						logger.Debug("BlockProducer: Paused while block pool is syncing")
						continue
					}
					newBlk := dpos.bp.ProduceBlock()
					if !dpos.Validate(newBlk) {
						logger.Error("DPoS: invalid block produced!")
						continue
					}
					dpos.updateNewBlock(newBlk)
				}
			case <-dpos.stopCh:
				return
			}
		}
	}()
}

func (dpos *DPOS) Stop() {
	logger.WithFields(logger.Fields{
		"peerid": dpos.node.GetPeerID(),
	}).Info("DPoS stops...")
	dpos.stopCh <- true
}

func (dpos *DPOS) hashAndSign(block *core.Block) {
	//block.SetNonce(0)
	hash := block.CalculateHash()
	block.SetHash(hash)
	ok := block.SignBlock(dpos.producerKey, hash)
	if !ok {
		logger.Warn("DPoS: Failed to sign the new block")
	}
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

	pubKeyHash, err := core.NewUserPubKeyHash(pubkey[1:])
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

// beneficiaryIsProducer is a requirement that ensures the reward is paid to the producer at the time slot
func (dpos *DPOS) beneficiaryIsProducer(block *core.Block) bool {
	if block == nil {
		logger.Debug("beneficiaryIsProducer requirement failed: block is empty")
		return false
	}

	producer := dpos.dynasty.ProducerAtATime(block.GetTimestamp())
	producerHash := core.HashAddress(core.NewAddress(producer))

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
	return !dpos.bp.IsIdle()
}

func (dpos *DPOS) updateNewBlock(newBlock *core.Block) {
	logger.WithFields(logger.Fields{
		"peerid": dpos.node.GetPeerID(),
		"height": newBlock.GetHeight(),
		"hash":   hex.EncodeToString(newBlock.GetHash()),
	}).Info("DPoS: Produced a new block")
	if !newBlock.VerifyHash() {
		logger.Warn("DPoS: Invalid hash in new block")
		return
	}
	err := dpos.bc.AddBlockToTail(newBlock)
	if err != nil {
		logger.Warn(err)
		return
	}
	dpos.node.BroadcastBlock(newBlock)
}
