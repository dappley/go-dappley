package block_producer

import (
	"github.com/dappley/go-dappley/core/block"
)

type Consensus interface {
	Validate(blk *block.Block) bool
	ProduceBlock(ProduceBlockFunc func(process func(*block.Block)))
}
