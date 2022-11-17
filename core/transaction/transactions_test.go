package transaction

import (
	"testing"

	"github.com/dappley/go-dappley/common"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestTransactions_FromProto(t *testing.T) {
	tx1 := Transaction{
		ID:       util.GenerateRandomAoB(1),
		Vin:      GenerateFakeTxInputs(),
		Vout:     GenerateFakeTxOutputs(),
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
	}

	tx2 := Transaction{
		ID:       util.GenerateRandomAoB(1),
		Vin:      GenerateFakeTxInputs(),
		Vout:     GenerateFakeTxOutputs(),
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
	}

	txs := Transactions{
		transactions: []Transaction{tx1, tx2},
	}

	pb := txs.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &transactionpb.Transactions{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	txs2 := Transactions{}
	txs2.FromProto(newpb)

	assert.Equal(t, txs, txs2)
}

func TestTransactions_GetTransactions(t *testing.T) {
	tx1 := Transaction{
		ID:       []byte{100},
		Vin:      GenerateFakeTxInputs(),
		Vout:     GenerateFakeTxOutputs(),
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
	}

	tx2 := Transaction{
		ID:       []byte{101},
		Vin:      GenerateFakeTxInputs(),
		Vout:     GenerateFakeTxOutputs(),
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
	}

	txs := NewTransactions([]Transaction{tx1, tx2})

	assert.Equal(t, []Transaction{tx1, tx2}, txs.GetTransactions())
}
