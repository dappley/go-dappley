package core

import (
	"testing"

	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

func TestTranstionHeapOperations(t *testing.T) {
	t1 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  generateFakeTxInputs(),
		Vout: generateFakeTxOutputs(),
		Tip:  5,
	}
	t2 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  generateFakeTxInputs(),
		Vout: generateFakeTxOutputs(),
		Tip:  10,
	}
	t3 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  generateFakeTxInputs(),
		Vout: generateFakeTxOutputs(),
		Tip:  2,
	}
	t4 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  generateFakeTxInputs(),
		Vout: generateFakeTxOutputs(),
		Tip:  20,
	}
	txPool := GetTxnPoolInstance()

	txPool.Push(t1)

	assert.Equal(t, 1, txPool.Len())

	txPool.Push(t2)
	assert.Equal(t, 2, txPool.Len())
	txPool.Push(t3)
	txPool.Push(t4)
	assert.Equal(t, 4, txPool.Len())

	for txPool.Len() > 0 {
		txPool.Pop()
	}
	assert.Equal(t, 0, txPool.Len())

}
