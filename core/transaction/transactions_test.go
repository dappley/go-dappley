package transaction

import (
	"testing"

	"github.com/dappley/go-dappley/core/account"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/transactionbase"

	"github.com/dappley/go-dappley/common"
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

func TestNewCoinbaseTX(t *testing.T) {
	t1 := NewCoinbaseTX(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 0, common.NewAmount(0))
	t2 := NewCoinbaseTX(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 0, common.NewAmount(0))
	expectVin := transactionbase.TXInput{nil, -1, []byte{0, 0, 0, 0, 0, 0, 0, 0}, []byte("Reward to 'dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB'")}
	expectVout := transactionbase.TXOutput{common.NewAmount(10000000), account.PubKeyHash([]byte{0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80, 0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49, 0xe5, 0x27, 0xf0, 0x42, 0x5d}), ""}
	assert.Equal(t, 1, len(t1.Vin))
	assert.Equal(t, expectVin, t1.Vin[0])
	assert.Equal(t, 1, len(t1.Vout))
	assert.Equal(t, expectVout, t1.Vout[0])
	assert.Equal(t, common.NewAmount(0), t1.Tip)

	// Assert that ltransaction.NewCoinbaseTX is deterministic (i.e. >1 coinbaseTXs in a block would have identical txid)
	assert.Equal(t, t1, t2)

	t3 := NewCoinbaseTX(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 1, common.NewAmount(0))

	assert.NotEqual(t, t1, t3)
	assert.NotEqual(t, t1.ID, t3.ID)
}
