package ltransaction

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/transaction"
)

func GenerateBlockWithCbtx(addr account.Address, lastblock *block.Block) *block.Block {
	//create a new block chain
	cbtx := NewCoinbaseTX(addr, "", lastblock.GetHeight(), common.NewAmount(0))
	b := block.NewBlock([]*transaction.Transaction{&cbtx}, lastblock, "")
	return b
}
