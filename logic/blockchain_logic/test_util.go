package blockchain_logic

import (
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/logic/utxo_logic"
)

func PrepareBlockContext(bc *Blockchain, blk *block.Block) *core.BlockContext {
	state := core.LoadScStateFromDatabase(bc.GetDb())
	utxoIndex := utxo_logic.NewUTXOIndex(bc.GetUtxoCache())
	utxoIndex.UpdateUtxoState(blk.GetTransactions())
	ctx := core.BlockContext{Block: blk, UtxoIndex: utxoIndex, State: state}
	return &ctx
}
