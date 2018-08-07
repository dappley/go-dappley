package core

import (
	"github.com/dappley/go-dappley/util"
	"time"
	"github.com/dappley/go-dappley/storage"
)

func GenerateMockBlock() *Block{
	bh1 := &BlockHeader{
		int32(1),
		[]byte("hash"),
		[]byte("prevhash"),
		1,
		time.Now().Unix(),
	}

	t1 := MockTransaction()
	t2 := MockTransaction()

	return &Block{
		header:       bh1,
		transactions: []*Transaction{t1,t2},
		height:       0,
	}
}

func GenerateMockBlockchain(size int) *Blockchain{
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s)

	for i:=0; i<size; i++{
		tailBlk, _ := bc.GetTailBlock()
		b:= NewBlock([]*Transaction{MockTransaction()},tailBlk)
		b.SetHash(b.CalculateHash())
		bc.UpdateNewBlock(b)
	}
	return bc
}

//the first item is the tail of the fork
func GenerateMockFork(size int, parent *Block) []*Block{
	fork := []*Block{}
	b := NewBlock(nil, parent)
	b.SetHash(b.CalculateHash())
	fork = append(fork, b)

	for i:=1; i<size; i++{
		b = NewBlock(nil, b)
		b.SetHash(b.CalculateHash())
		fork = append([]*Block{b}, fork...)
	}
	return fork
}

//the first item is the tail of the fork
func GenerateMockForkWithValidTx(size int, parent *Block) []*Block{
	fork := []*Block{}
	b := NewBlock(nil, parent)
	b.SetHash(b.CalculateHash())
	fork = append(fork, b)

	for i:=1; i<size; i++{
		b = NewBlock([]*Transaction{MockTransaction()}, b)
		b.SetHash(b.CalculateHash())
		fork = append([]*Block{b}, fork...)
	}
	return fork
}

//the first item is the tail of the fork
func GenerateMockForkWithInvalidTx(size int, parent *Block) []*Block{
	fork := []*Block{}
	b := NewBlock(nil, parent)
	b.SetHash(b.CalculateHash())
	fork = append(fork, b)

	for i:=1; i<size; i++{
		b = NewBlock([]*Transaction{MockTransaction()}, b)
		b.SetHash(b.CalculateHash())
		fork = append([]*Block{b}, fork...)
	}
	return fork
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

func GenerateMockTransactionPool(numOfTxs int) *TransactionPool{
	txPool := &TransactionPool{}
	for i := 0; i < numOfTxs; i++ {
		txPool.Push(*MockTransaction())
	}
	return txPool
}