package core

import (
	"fmt"
	"testing"

	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/gogo/protobuf/proto"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func generateFakeTxInputs() []TXInput {

	var a = []TXInput{
		TXInput{getAoB(2), 10, getAoB(2), getAoB(2)},
		TXInput{getAoB(2), 5, getAoB(2), getAoB(2)},
	}
	return a
}

func generateFakeTxOutputs() []TXOutput {

	var a = []TXOutput{
		TXOutput{1, getAoB(2)},
		TXOutput{2, getAoB(2)},
	}
	return a
}

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

	//heap.Init(h)
	txPool.Push(t1)

	assert.Equal(t, 1, txPool.Len())

	txPool.Push(t2)
	assert.Equal(t, 2, txPool.Len())
	txPool.Push(t3)
	txPool.Push(t4)
	assert.Equal(t, 4, txPool.Len())
	//var test_slice = []Transaction{}
	for txPool.Len() > 0 {
		var txnInterface = txPool.Pop()
		fmt.Println(txnInterface.(Transaction).Tip)
	}
	assert.Equal(t, 0, txPool.Len())

}

func TestTransaction_Proto(t *testing.T) {
	t1 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  generateFakeTxInputs(),
		Vout: generateFakeTxOutputs(),
		Tip:  5,
	}

	pb := t1.ToProto()
	mpb,err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.Transaction{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	t2 := Transaction{}
	t2.FromProto(newpb)

	assert.Equal(t,t1,t2)
}
