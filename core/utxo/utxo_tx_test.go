package utxo

import (
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/stretchr/testify/assert"
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

func TestUTXOTx_DeepCopy(t *testing.T) {
	expectedUtxoTxIndices := map[string]*UTXO{"test_0": utxo1, "test_1": utxo2}
	utxoTx := NewUTXOTx()
	// copy empty utxoTx
	newUtxoTx := utxoTx.DeepCopy()
	assert.Equal(t, map[string]*UTXO{}, newUtxoTx.Indices)
	// copy utxoTx containing utxo1 and utxo2
	utxoTx.PutUtxo(utxo1)
	utxoTx.PutUtxo(utxo2)
	assert.Equal(t, expectedUtxoTxIndices, utxoTx.Indices)
	newUtxoTx = utxoTx.DeepCopy()
	assert.Equal(t, expectedUtxoTxIndices, newUtxoTx.Indices)
	// removing from original utxoTx should not change the copy
	utxoTx.RemoveUtxo(utxo1.Txid, 0)
	assert.Equal(t, expectedUtxoTxIndices, newUtxoTx.Indices)
}

func TestStringEntry_Hash(t *testing.T) {
	se1 := StringEntry("hello")
	se2 := StringEntry("world")

	assert.Equal(t, uint32(0x4f9f2cab), se1.Hash())
	assert.Equal(t, uint32(0x37a3e893), se2.Hash())
}

func TestStringEntry_Equal(t *testing.T) {
	se1 := StringEntry("string1")
	se2 := StringEntry("string1")
	se3 := StringEntry("string2")

	assert.True(t, se1.Equal(&se2))
	assert.False(t, se1.Equal(&se3))
}

func TestSyncPool_Free(t *testing.T) {
	utxoTx := NewUTXOTx()
	utxoTx.PutUtxo(utxo1)
	utxoTx.PutUtxo(utxo2)
	indices := utxoTx.Indices
	assert.NotNil(t, indices[utxo1.GetUTXOKey()])
	assert.NotNil(t, indices[utxo2.GetUTXOKey()])

	Free(&utxoTx)
	assert.Nil(t, indices[utxo1.GetUTXOKey()])
	assert.Nil(t, indices[utxo2.GetUTXOKey()])
}

func TestSyncPool_NewSyncPool(t *testing.T) {
	testPool := NewSyncPool(10, 100, 2)
	assert.Equal(t, 4, len(testPool.classes))
	assert.Equal(t, 4, len(testPool.classesSize))
	assert.Equal(t, 10, testPool.classesSize[0])
	assert.NotNil(t, testPool.classes[0])
	assert.Equal(t, 20, testPool.classesSize[1])
	assert.NotNil(t, testPool.classes[1])
	assert.Equal(t, 40, testPool.classesSize[2])
	assert.NotNil(t, testPool.classes[2])
	assert.Equal(t, 80, testPool.classesSize[3])
	assert.NotNil(t, testPool.classes[3])
}
