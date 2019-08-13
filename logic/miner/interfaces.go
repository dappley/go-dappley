package miner

import (
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/block"
)

type Consensus interface {
	GetBlockProduceNotifier() chan bool
	Validate(*block.Block) bool
	GetProcess() consensus.Process
}
