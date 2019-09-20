package block

import (
	"github.com/dappley/go-dappley/core/transaction"
	"time"
)

func GenerateMockBlock() *Block {
	t1 := transaction.MockTransaction()
	t2 := transaction.MockTransaction()

	return NewBlockWithRawInfo(
		[]byte("hash"),
		[]byte("prevhash"),
		1,
		time.Now().Unix(),
		0,
		[]*transaction.Transaction{t1, t2},
	)
}
