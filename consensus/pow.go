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
	"math"
	"math/big"

	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/logic/block_logic"
	"github.com/dappley/go-dappley/logic/blockchain_logic"

	logger "github.com/sirupsen/logrus"
)

const defaultTargetBits = 0

var maxNonce int64 = math.MaxInt64

type ProofOfWork struct {
	miner  *logic.BlockProducerLogic
	target *big.Int
	bm     *blockchain_logic.BlockchainManager
	stopCh chan bool
}

func NewProofOfWork() *ProofOfWork {
	p := &ProofOfWork{
		miner:  logic.NewBlockProducerLogic(),
		stopCh: make(chan bool, 1),
	}
	p.SetTargetBit(defaultTargetBits)
	return p
}

func (pow *ProofOfWork) Setup(cbAddr string, bm *blockchain_logic.BlockchainManager) {
	pow.bm = bm

	var bc *blockchain_logic.Blockchain
	if pow.bm != nil {
		bc = bm.Getblockchain()
	}

	pow.miner.Setup(bc, cbAddr)
	pow.miner.SetProcess(pow.calculateValidHash)
}

func (pow *ProofOfWork) SetTargetBit(bit int) {
	if bit < 0 || bit > 256 {
		return
	}
	target := big.NewInt(1)
	pow.target = target.Lsh(target, uint(256-bit))
}

func (pow *ProofOfWork) SetKey(key string) {
	// pow does not require block signing
}

func (pow *ProofOfWork) GetProducerAddress() string {
	return pow.miner.Beneficiary()
}

func (pow *ProofOfWork) Start() {
	logger.Info("PoW starts...")
	pow.resetStopCh()
	go pow.mineBlocks()
}

func (pow *ProofOfWork) Stop() {
	logger.Info("PoW stops...")
	pow.stopCh <- true
}

func (pow *ProofOfWork) mineBlocks() {
	logger.Info("PoW: mining starts.")
	for {
		select {
		case <-pow.stopCh:
			logger.Info("PoW: mining stopped.")
			return
		default:
			if pow.bm.Getblockchain().GetState() != blockchain.BlockchainReady {
				logger.Debug("BlockProducerInfo: Paused while block pool is syncing")
				continue
			}
			newBlock := pow.miner.ProduceBlock(0)
			if newBlock == nil || !pow.Validate(newBlock.Block) {
				logger.WithFields(logger.Fields{"block": newBlock}).Debug("PoW: the block mined is invalid.")
				pow.miner.BlockProduceFinish()
				return
			}
			pow.updateNewBlock(newBlock)
			pow.miner.BlockProduceFinish()
		}
	}
}

func (pow *ProofOfWork) resetStopCh() {
L:
	for {
		select {
		case <-pow.stopCh:
		default:
			break L
		}
	}
}

func (pow *ProofOfWork) calculateValidHash(ctx *blockchain_logic.BlockContext) {
	for {
		select {
		case <-pow.stopCh:
			pow.stopCh <- true
			return
		default:
			hash := block_logic.CalculateHashWithNonce(ctx.Block)
			ctx.Block.SetHash(hash)
			if !pow.isHashBelowTarget(ctx.Block) {
				pow.tryDifferentNonce(ctx.Block)
				continue
			}
			return
		}
	}

}

func (pow *ProofOfWork) IsProducingBlock() bool {
	return !pow.miner.IsIdle()
}

func (pow *ProofOfWork) isHashBelowTarget(block *block.Block) bool {
	var hashInt big.Int

	hash := block.GetHash()
	hashInt.SetBytes(hash)

	return hashInt.Cmp(pow.target) == -1
}

func (pow *ProofOfWork) Validate(block *block.Block) bool {
	return pow.isHashBelowTarget(block)
}

func (pow *ProofOfWork) tryDifferentNonce(block *block.Block) {
	nonce := block.GetNonce()
	if nonce >= maxNonce {
		logger.Warn("PoW: tried all possible nonce.")
	}
	block.SetNonce(nonce + 1)
}

func (pow *ProofOfWork) updateNewBlock(ctx *blockchain_logic.BlockContext) {
	logger.WithFields(logger.Fields{"height": ctx.Block.GetHeight()}).Info("PoW: minted a new block.")
	if !block_logic.VerifyHash(ctx.Block) {
		logger.Warn("PoW: the new block contains invalid hash (mining might have been interrupted).")
		return
	}
	err := pow.bm.Getblockchain().AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}

	pow.bm.BroadcastBlock(ctx.Block)
}

func (pow *ProofOfWork) AddProducer(producer string) error {
	return nil
}

func (pow *ProofOfWork) GetProducers() []string {
	return nil
}

func (pow *ProofOfWork) Produced(blk *block.Block) bool {
	if blk != nil {
		return pow.miner.Produced(blk)
	}
	return false
}

func (pow *ProofOfWork) CheckLibPolicy(b *block.Block) (*block.Block, bool) {
	return nil, true
}
