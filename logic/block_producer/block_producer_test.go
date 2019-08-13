package block_producer

import (
	"testing"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/logic/transaction_pool"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

func TestBlockProducerInfo_ProduceBlock(t *testing.T) {
	bp := NewBlockProducer()
	cbAddr := "1FoupuhmPN4q1wiUrM5QaYZjYKKLLXzPPg"
	bc := blockchain_logic.CreateBlockchain(
		account.NewAddress(cbAddr),
		storage.NewRamStorage(),
		nil,
		transaction_pool.NewTransactionPool(nil, 128),
		nil,
		100000,
	)
	bp.Setup(bc, cbAddr)
	processRuns := false
	bp.SetProcess(func(ctx *blockchain_logic.BlockContext) {
		processRuns = true
	})
	block := bp.produceBlock(0)
	assert.True(t, processRuns)
	assert.NotNil(t, block)
}
