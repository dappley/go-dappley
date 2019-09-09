package blockproducer

import "github.com/dappley/go-dappley/metrics"

var (
	TxAddToBlockCost = metrics.NewHistogram("tx.AddToBlock.cost")
)
