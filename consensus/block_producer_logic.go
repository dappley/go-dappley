package consensus

import "github.com/dappley/go-dappley/logic/blockchain_logic"

// process defines the procedure to produce a valid block modified from a raw (unhashed/unsigned) block
type process func(ctx *blockchain_logic.BlockContext)

type BlockProducerLogic struct {
	bc          *blockchain_logic.Blockchain
	beneficiary string
	process     process
	idle        bool
}
