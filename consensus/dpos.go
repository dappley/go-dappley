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
	"fmt"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/util"
	"github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"
	"strings"
	"time"
)

const version = byte(0x00)
const addressChecksumLen = 4

type Dpos struct {
	bc        *core.Blockchain
	miner     *Miner
	mintBlkCh chan (*MinedBlock)
	node      core.NetService
	quitCh    chan (bool)
	dynasty   *Dynasty
	slot	*lru.Cache
}

func NewDpos() *Dpos {
	dpos := &Dpos{
		miner:     NewMiner(),
		mintBlkCh: make(chan (*MinedBlock), 1),
		node:      nil,
		quitCh:    make(chan (bool), 1),
	}

	slot, err := lru.New(128)
	if err != nil {
		logger.Panic(err)
	}
	dpos.slot = slot
	return dpos
}

func (dpos *Dpos) Setup(node core.NetService, cbAddr string) {
	dpos.bc = node.GetBlockchain()
	dpos.node = node
	dpos.miner.Setup(dpos.bc, cbAddr, dpos.mintBlkCh)
}

func (dpos *Dpos) SetTargetBit(bit int) {
	dpos.miner.SetTargetBit(bit)
}

func (dpos *Dpos) SetKey(key string) {
	dpos.miner.SetPrivKey(key)
}

func (dpos *Dpos) SetDynasty(dynasty *Dynasty) {
	dpos.dynasty = dynasty
}

func (dpos *Dpos) GetDynasty() *Dynasty {
	return dpos.dynasty
}

func (dpos *Dpos) GetBlockChain() *core.Blockchain {
	return dpos.bc
}


func (dpos *Dpos) Validate(block *core.Block) bool{
	pass := dpos.miner.Validate(block) && dpos.dynasty.ValidateProducer(block)
	if pass {
		dpos.slot.Add(block.GetTimestamp(), block)
	}
	return pass
}

func (dpos *Dpos) Start() {
	go func() {
		logger.Info("Dpos Starts...", dpos.node.GetPeerID())
		ticker := time.NewTicker(time.Second).C
		for {
			select {
			case now := <-ticker:
				if dpos.dynasty.IsMyTurn(dpos.miner.cbAddr, now.Unix()) {
					logger.Info("Dpos: My Turn to Mint! I am ", dpos.node.GetPeerID())
					dpos.miner.Start()
				}
			case minedBlk := <-dpos.mintBlkCh:
				if minedBlk.isValid {
					logger.Info("Dpos: A Block has been mined! ", dpos.node.GetPeerID())
					dpos.updateNewBlock(minedBlk.block)
				}
			case <-dpos.quitCh:
				logger.Info("Dpos: Dpos Stops! ", dpos.node.GetPeerID())
				return
			}
		}
	}()
}

func (dpos *Dpos) Stop() {
	dpos.quitCh <- true
	dpos.miner.Stop()
}


func (dpos *Dpos) CheckDoubleMint(block *core.Block) bool {
	if preBlock, exist := dpos.slot.Get(block.GetTimestamp()); exist {
		if !core.IsHashEqual(preBlock.(*core.Block).GetHash(), block.GetHash()) {
			logger.Warn("Someone is trying to mint multiple blocks at the same time!")
			return true
		}
	}
	return false
}
func (dpos *Dpos) StartNewBlockMinting(){
	dpos.miner.Stop()
}
func (dpos *Dpos) FullyStop() bool {
	v := <-dpos.miner.exitCh
	return v
}

func (dpos *Dpos) updateNewBlock(newBlock *core.Block) {
	logger.Info("DPoS: Minted a new block. height:", newBlock.GetHeight())
	dpos.bc.AddBlockToTail(newBlock)
	dpos.node.BroadcastBlock(newBlock)
}

func GenerateAddress(pubkey []byte) string {

	pubKeyHash, _ := core.HashPubKey(pubkey[1:])

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := core.Checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := util.Base58Encode(fullPayload)
	//15KciXJD9vLhhJQjqDuAgPs83r7sCi9YYK

	return string(fmt.Sprintf("%s", address))
}

func (dpos *Dpos) VerifyBlock(block *core.Block) bool {
	hash1 := block.GetHash()
	sign := block.GetSign()

	producer := dpos.dynasty.ProducerAtATime(block.GetTimestamp())

	if hash1 == nil {
		logger.Warn("DPoS: block hash empty!")
		return false
	}
	if sign == nil {
		logger.Warn("DPoS: block signature empty!")
		return false
	}

	pubkey, err := secp256k1.RecoverECDSAPublicKey(hash1, sign)
	if err != nil {
		logger.Warn("DPoS: Get pub key from block signature error!")
		return false
	}

	address := GenerateAddress(pubkey)

	if strings.Compare(address, producer) == 0 {
		return true
	}

	return false

}
