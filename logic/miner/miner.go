package miner

import (
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/block_logic"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	logger "github.com/sirupsen/logrus"
	"time"
)

const maxMintingTimeInMs = 2000
const NanoSecsInMilliSec = 1000000

type Miner struct {
	con      Consensus
	producer *consensus.BlockProducer
	bc       *blockchain_logic.Blockchain
}

func NewMiner(con Consensus, bc *blockchain_logic.Blockchain) *Miner {
	return &Miner{
		con:      con,
		producer: consensus.NewBlockProducer(),
		bc:       bc,
	}
}

func (miner *Miner) Start() {
	go func() {
		logger.Info("Miner Starts...")
		for {
			select {
			case <-miner.con.GetBlockProduceNotifier():
				deadlineInMs := time.Now().UnixNano()/NanoSecsInMilliSec + maxMintingTimeInMs

				logger.Infof("Miner: producing block... ***time is %v***", time.Now().Unix())

				// Do not produce block if block pool is syncing
				if miner.bc.GetState() != blockchain.BlockchainReady {
					logger.Info("DPoS: block producer paused because block pool is syncing.")
					continue
				}
				ctx := miner.producer.ProduceBlock(deadlineInMs)
				if ctx == nil || !miner.con.Validate(ctx.Block) {
					miner.producer.BlockProduceFinish()
					logger.Error("DPoS: produced an invalid block!")
					continue
				}
				miner.updateNewBlock(ctx)
				miner.producer.BlockProduceFinish()
			}
		}
	}()
}

func (miner *Miner) updateNewBlock(ctx *blockchain_logic.BlockContext) {
	logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	}).Info("DPoS: produced a new block.")
	if !block_logic.VerifyHash(ctx.Block) {
		logger.Warn("DPoS: hash of the new block is invalid.")
		return
	}

	// TODO Refactoring lib calculate position, check lib when create BlockContext instance
	lib, ok := miner.con.CheckLibPolicy(ctx.Block)
	if !ok {
		logger.Warn("DPoS: the number of producers is not enough.")
		tailBlock, _ := miner.bc.GetTailBlock()
		miner.BroadcastBlock(tailBlock)
		return
	}
	ctx.Lib = lib

	err := miner.bc.AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}
	miner.BroadcastBlock(ctx.Block)
}
