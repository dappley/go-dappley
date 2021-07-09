package utxo

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/stretchr/testify/assert"
	"testing"
)

var utxo1 = &UTXO{
	TXOutput: transactionbase.TXOutput{
		Value:      common.NewAmount(5),
		PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
		Contract:   "contract",
	},
	Txid:     []byte{0x74, 0x65, 0x73, 0x74},
	TxIndex:  0,
	UtxoType: UtxoNormal,
}
var utxo2 = &UTXO{
	TXOutput: transactionbase.TXOutput{
		Value:      common.NewAmount(10),
		PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6d},
		Contract:   "contract",
	},
	Txid:     []byte{0x74, 0x65, 0x73, 0x74},
	TxIndex:  1,
	UtxoType: UtxoNormal,
}

func TestNewUTXOTx(t *testing.T) {
	assert.Equal(t, UTXOTx{Indices: map[string]*UTXO{}}, NewUTXOTx())
}

func TestNewUTXOTxWithData(t *testing.T) {
	expected := UTXOTx{map[string]*UTXO{"test_0": utxo1}}

	assert.Equal(t, expected, NewUTXOTxWithData(utxo1))
}

func TestUTXOTx_PutUtxo(t *testing.T) {
	utxoTx := NewUTXOTx()
	utxoTx.PutUtxo(utxo1)

	expected := UTXOTx{map[string]*UTXO{"test_0": utxo1}}
	assert.Equal(t, expected, utxoTx)
}

func TestUTXOTx_GetUtxo(t *testing.T) {
	utxoTx := NewUTXOTx()
	// get non-existent key
	assert.Nil(t, utxoTx.GetUtxo(utxo1.Txid, 0))

	utxoTx.PutUtxo(utxo1)
	utxoTx.PutUtxo(utxo2)

	assert.Equal(t, utxo1, utxoTx.GetUtxo(utxo1.Txid, 0))
	assert.Equal(t, utxo2, utxoTx.GetUtxo(utxo2.Txid, 1))
}

func TestUTXOTx_RemoveUtxo(t *testing.T) {
	utxoTx := NewUTXOTx()
	utxoTx.PutUtxo(utxo1)
	utxoTx.PutUtxo(utxo2)

	assert.Equal(t, UTXOTx{map[string]*UTXO{"test_0": utxo1, "test_1": utxo2}}, utxoTx)
	utxoTx.RemoveUtxo(utxo1.Txid, 0)
	assert.Equal(t, UTXOTx{map[string]*UTXO{"test_1": utxo2}}, utxoTx)
	utxoTx.RemoveUtxo(utxo2.Txid, 1)
	assert.Equal(t, UTXOTx{map[string]*UTXO{}}, utxoTx)
}

func TestUTXOTx_Size(t *testing.T) {
	utxoTx := NewUTXOTx()
	assert.Equal(t, 0, utxoTx.Size())
	utxoTx.PutUtxo(utxo1)
	utxoTx.PutUtxo(utxo2)
	assert.Equal(t, 2, utxoTx.Size())
	utxoTx.RemoveUtxo(utxo1.Txid, 0)
	assert.Equal(t, 1, utxoTx.Size())
}

func TestUTXOTx_GetAllUtxos(t *testing.T) {
	utxoTx1 := NewUTXOTx()
	utxoTx1.PutUtxo(utxo1)
	utxoTx1.PutUtxo(utxo2)

	utxos := utxoTx1.GetAllUtxos()
	utxoTx2 := NewUTXOTx()
	// since the utxos are added in random order, we test by adding them back into a map
	for _, utxo := range utxos {
		utxoTx2.PutUtxo(utxo)
	}
	assert.Equal(t, utxoTx1.Indices, utxoTx2.Indices)
}

func TestUTXOTx_PrepareUtxos(t *testing.T) {
	tests := []struct {
		name           string
		utxoTx         UTXOTx
		amount         *common.Amount
		expectedResult []*UTXO
		expectedOk     bool
	}{
		{
			name:           "empty utxoTx",
			utxoTx:         UTXOTx{map[string]*UTXO{}},
			amount:         common.NewAmount(0),
			expectedResult: nil,
			expectedOk:     false,
		},
		{
			name:           "success",
			utxoTx:         UTXOTx{map[string]*UTXO{"test_0": utxo1, "test_1": utxo2}},
			amount:         common.NewAmount(15),
			expectedResult: []*UTXO{utxo1, utxo2},
			expectedOk:     true,
		},
		{
			name:           "amount too high",
			utxoTx:         UTXOTx{map[string]*UTXO{"test_0": utxo1, "test_1": utxo2}},
			amount:         common.NewAmount(16),
			expectedResult: nil,
			expectedOk:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := tt.utxoTx.PrepareUtxos(tt.amount)
			if tt.expectedOk {
				assert.True(t, ok)
				// output of PrepareUtxos is in random order so I compare using maps
				expectedUtxoMap := map[string]*UTXO{}
				for _, utxo := range tt.expectedResult {
					expectedUtxoMap[utxo.GetUTXOKey()] = utxo
				}
				resultUtxoMap := map[string]*UTXO{}
				for _, utxo := range result {
					resultUtxoMap[utxo.GetUTXOKey()] = utxo
				}
				assert.Equal(t, expectedUtxoMap, resultUtxoMap)
			} else {
				assert.Nil(t, result)
				assert.False(t, ok)
			}
		})
	}
}

func TestUTXOTx_DeepCopy(t *testing.T) {
	utxoTx := NewUTXOTx()
	newUtxoTx := utxoTx.DeepCopy()
	assert.Equal(t, map[string]*UTXO{}, newUtxoTx.Indices)
	utxoTx.PutUtxo(utxo1)
	utxoTx.PutUtxo(utxo2)
	newUtxoTx = utxoTx.DeepCopy()
	assert.Equal(t, utxoTx.Indices, newUtxoTx.Indices)
}

func TestStringEntry_Hash(t *testing.T) {
	se1 := StringEntry("hello")
	se2 := StringEntry("world")

	assert.Equal(t, uint32(0x4f9f2cab), se1.Hash())
	assert.Equal(t, uint32(0x37a3e893), se2.Hash())
}
