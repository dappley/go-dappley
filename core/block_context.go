package core

import "github.com/dappley/go-dappley/core/block"

type BlockContext struct {
	Block     *block.Block
	Lib       *block.Block
	UtxoIndex *UTXOIndex
	State     *ScState
}
