package blockchain_logic

import (
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/logic/utxo_logic"
)

type BlockContext struct {
	Block     *block.Block
	Lib       *block.Block
	UtxoIndex *utxo_logic.UTXOIndex
	State     *core.ScState
}
