package utxo

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewUTXOTx(t *testing.T) {
	assert.Equal(t, UTXOTx{Indices: map[string]*UTXO{}}, NewUTXOTx())
}

func TestNewUTXOTxWithData(t *testing.T) {
	utxo := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:     []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:  0,
		UtxoType: UtxoNormal,
	}
	expected := UTXOTx{map[string]*UTXO{"test_0": utxo}}

	assert.Equal(t, expected, NewUTXOTxWithData(utxo))
}

func TestStringEntry_Hash(t *testing.T) {
	se1 := StringEntry("hello")
	se2 := StringEntry("world")

	assert.Equal(t, uint32(0x4f9f2cab), se1.Hash())
	assert.Equal(t, uint32(0x37a3e893), se2.Hash())
}
