package block_producer

import (
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/block"
)

type Consensus interface {
	ShouldProduceBlock(producerAddr string, currTime int64) bool
	GetBlockProduceNotifier() chan bool
	Validate(blk *block.Block) bool
	GetProcess() consensus.Process
	Start()
	Stop()
	ProduceBlock(ProduceBlockFunc func(process func(*block.Block)))
}
