package core

import (
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/logic/utxo_logic"
)

type BlockContext struct {
	Block     *block.Block
	Lib       *block.Block
	UtxoIndex *utxo_logic.UTXOIndex
	State     *ScState
}
