package core

import (
	"container/heap"
	"testing"

	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

var t1 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  generateFakeTxInputs(),
	Vout: generateFakeTxOutputs(),
	Tip:  2,
}
var t2 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  generateFakeTxInputs(),
	Vout: generateFakeTxOutputs(),
	Tip:  5,
}
var t3 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  generateFakeTxInputs(),
	Vout: generateFakeTxOutputs(),
	Tip:  10,
}
var t4 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  generateFakeTxInputs(),
	Vout: generateFakeTxOutputs(),
	Tip:  20,
}

var expectPopOrder = []int64{20, 10, 5, 2}

var popInputOrder = []struct {
	order []Transaction
}{
	{[]Transaction{t4, t3, t2, t1}},
	{[]Transaction{t1, t2, t3, t4}},
	{[]Transaction{t2, t1, t4, t3}},
	{[]Transaction{t4, t1, t3, t2}},
}

//transaction pool push function
func TestTxPoolPush(t *testing.T) {
	txPool := GetTxnPoolInstance()
	heap.Push(txPool, t1)
	assert.Equal(t, 1, txPool.Len())
	heap.Push(txPool, t2)
	assert.Equal(t, 2, txPool.Len())
	heap.Push(txPool, t3)
	heap.Push(txPool, t4)
	assert.Equal(t, 4, txPool.Len())
}

func TestTranstionPoolPop(t *testing.T) {
	for _, tt := range popInputOrder {
		var popOrder = []int64{}
		txPool := GetTxnPoolInstance()
		for _, t := range tt.order {
			heap.Push(txPool, t)
		}
		for txPool.Len() > 0 {
			popOrder = append(popOrder, heap.Pop(txPool).(Transaction).Tip)
		}
		assert.Equal(t, expectPopOrder, popOrder)
	}

}