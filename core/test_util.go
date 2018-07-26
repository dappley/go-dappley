package core

import (
	"github.com/dappley/go-dappley/util"
	"time"
)

func GenerateMockBlock() *Block{
	bh1 := &BlockHeader{
		[]byte("hash"),
		[]byte("prevhash"),
		1,
		time.Now().Unix(),
	}

	t1 := MockTransaction()

	return &Block{
		header:       bh1,
		transactions: []*Transaction{t1},
		height:       0,
	}
}

func MockTransaction() *Transaction{
	return &Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  MockTxInputs(),
		Vout: MockTxOutputs(),
		Tip:  5,
	}
}

func MockTxInputs() []TXInput {
	return []TXInput{
		{util.GenerateRandomAoB(2),
			6,
			util.GenerateRandomAoB(2),
			util.GenerateRandomAoB(2)},
		{util.GenerateRandomAoB(2),
			2,
			util.GenerateRandomAoB(2),
			util.GenerateRandomAoB(2)},
	}
}

func MockTxOutputs() []TXOutput {
	return []TXOutput{
		{5, util.GenerateRandomAoB(2)},
		{7, util.GenerateRandomAoB(2)},
	}
}

