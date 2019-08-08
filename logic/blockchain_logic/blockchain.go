package blockchain_logic

import (
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/block_logic"
)

func GetMaxHeight(bc *blockchain.Blockchain, db Storage) uint64 {
	tailHash := bc.GetTailBlockHash()
	blk, err := block_logic.GetBlockByHash(tailHash, db)
	if err != nil {
		return 0
	}
	return blk.GetHeight()
}
