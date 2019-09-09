package blockproducer

import (
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/block"
)

type Consensus interface {
	ShouldProduceBlock(producerAddr string, currTime int64) bool
	Validate(blk *block.Block) bool
	GetProcess() consensus.Process
	ProduceBlock(ProduceBlockFunc func(process func(*block.Block)))
}
