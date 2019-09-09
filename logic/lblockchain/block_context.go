package lblockchain

import (
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/logic/lutxo"
)

type BlockContext struct {
	Block     *block.Block
	UtxoIndex *lutxo.UTXOIndex
	State     *scState.ScState
}
