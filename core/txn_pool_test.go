package core

import (
	"testing"

	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

var t1 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  2,
}
var t2 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  5,
}
var t3 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  10,
}
var t4 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  20,
}

var expectPopOrder = []uint64{20, 10, 5, 2}

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
	txPool.Transactions.StructPush(t1)
	assert.Equal(t, 1, txPool.Transactions.Len())
	txPool.Transactions.StructPush(t2)
	assert.Equal(t, 2, txPool.Transactions.Len())
	txPool.Transactions.StructPush(t3)
	txPool.Transactions.StructPush(t4)
	assert.Equal(t, 4, txPool.Transactions.Len())
	cleanUpPool()
}

func TestTranstionPoolPop(t *testing.T) {
	for _, tt := range popInputOrder {
		var popOrder = []uint64{}
		txPool := GetTxnPoolInstance()
		for _, tx := range tt.order {
			txPool.Transactions.StructPush(tx)
		}
		for txPool.Transactions.Len() > 0 {
			popOrder = append(popOrder, txPool.Transactions.PopRight().(Transaction).Tip)
		}
		assert.Equal(t, expectPopOrder, popOrder)
	}
}

func cleanUpPool() {
	txPool := GetTxnPoolInstance()
	for txPool.Transactions.Len() > 0 {
		txPool.Transactions.PopRight()
	}
}