// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"testing"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

var bh1 = &BlockHeader{
	[]byte("hash"),
	nil,
	1,
	time.Now().Unix(),
	nil,
	0,
}

var bh2 = &BlockHeader{
	[]byte("hash1"),
	[]byte("hash"),
	1,
	time.Now().Unix(),
	nil,
	1,
}

// Padding address to 32 Byte
var address1Bytes = []byte("address1000000000000000000000000")
var address2Bytes = []byte("address2000000000000000000000000")
var address1Hash, _ = HashPubKey(address1Bytes)
var address2Hash, _ = HashPubKey(address2Bytes)

func GenerateUtxoMockBlockWithoutInputs() *Block {

	t1 := MockUtxoTransactionWithoutInputs()
	return &Block{
		header:       bh1,
		transactions: []*Transaction{t1},
	}
}

func GenerateUtxoMockBlockWithInputs() *Block {

	t1 := MockUtxoTransactionWithInputs()
	return &Block{
		header:       bh2,
		transactions: []*Transaction{t1},
	}
}

func MockUtxoTransactionWithoutInputs() *Transaction {
	return &Transaction{
		ID:   []byte("tx1"),
		Vin:  []TXInput{},
		Vout: MockUtxoOutputsWithoutInputs(),
		Tip:  5,
	}
}

func MockUtxoTransactionWithInputs() *Transaction {
	return &Transaction{
		ID:   []byte("tx2"),
		Vin:  MockUtxoInputs(),
		Vout: MockUtxoOutputsWithInputs(),
		Tip:  5,
	}
}

func MockUtxoInputs() []TXInput {
	return []TXInput{
		{
			[]byte("tx1"),
			0,
			util.GenerateRandomAoB(2),
			address1Bytes},
		{
			[]byte("tx1"),
			1,
			util.GenerateRandomAoB(2),
			address1Bytes},
	}
}

func MockUtxoOutputsWithoutInputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(5), address1Hash},
		{common.NewAmount(7), address1Hash},
	}
}

func MockUtxoOutputsWithInputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(4), address1Hash},
		{common.NewAmount(5), address2Hash},
		{common.NewAmount(3), address2Hash},
	}
}

func TestAddUTXO(t *testing.T) {
	db :=  storage.NewRamStorage()
	defer db.Close()

	txout := TXOutput{common.NewAmount(5), address1Hash}
	utxoIndex := make(UTXOIndex)

	utxoIndex.addUTXO(txout, []byte{1}, 0)

	addr1UTXOs := utxoIndex[string(address1Hash)]
	assert.Equal(t, 1, len(addr1UTXOs))
	assert.Equal(t, txout.Value, addr1UTXOs[0].Value)
	assert.Equal(t, []byte{1}, addr1UTXOs[0].Txid)
	assert.Equal(t, 0, addr1UTXOs[0].TxIndex)

	addr2UTXOs := utxoIndex["address2"]
	assert.Equal(t, 0, len(addr2UTXOs))
}

func TestRemoveUTXO(t *testing.T){
	db :=  storage.NewRamStorage()
	defer db.Close()

	utxoIndex := make(UTXOIndex)

	utxoIndex[string(address1Hash)] = append(utxoIndex[string(address1Hash)], &UTXO{common.NewAmount(5), address1Hash, []byte{1}, 0})
	utxoIndex[string(address1Hash)] = append(utxoIndex[string(address1Hash)], &UTXO{common.NewAmount(2), address1Hash, []byte{1}, 1})
	utxoIndex[string(address1Hash)] = append(utxoIndex[string(address1Hash)], &UTXO{common.NewAmount(2), address1Hash, []byte{2}, 0})
	utxoIndex[string(address2Hash)] = append(utxoIndex[string(address2Hash)], &UTXO{common.NewAmount(4), address2Hash, []byte{1}, 2})

	err := utxoIndex.removeUTXO([]byte{1}, 0)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(utxoIndex[string(address1Hash)]))
	assert.Equal(t, 1, len(utxoIndex[string(address2Hash)]))

	err = utxoIndex.removeUTXO([]byte{2}, 1)  // Does not exists

	assert.NotNil(t, err)
	assert.Equal(t, 2, len(utxoIndex[string(address1Hash)]))
	assert.Equal(t, 1, len(utxoIndex[string(address2Hash)]))
}

func TestUpdate(t *testing.T) {
	db :=  storage.NewRamStorage()
	defer db.Close()

	blk := GenerateUtxoMockBlockWithoutInputs()
	utxoIndex := make(UTXOIndex)
	utxoIndex.BuildForkUtxoIndex(blk, db)
	utxoIndexInDB := LoadUTXOIndex(db)

	// Assert that both the original instance and the database copy are updated correctly
	for _, index := range []UTXOIndex{utxoIndex, utxoIndexInDB} {
		assert.Equal(t, 2, len(index[string(address1Hash)]))
		assert.Equal(t, blk.transactions[0].ID, index[string(address1Hash)][0].Txid)
		assert.Equal(t, 0, index[string(address1Hash)][0].TxIndex)
		assert.Equal(t, blk.transactions[0].Vout[0].Value, index[string(address1Hash)][0].Value)
		assert.Equal(t, blk.transactions[0].ID, index[string(address1Hash)][1].Txid)
		assert.Equal(t, 1, index[string(address1Hash)][1].TxIndex)
		assert.Equal(t, blk.transactions[0].Vout[1].Value, index[string(address1Hash)][1].Value)
	}
}

func TestUpdate_Failed(t *testing.T) {
	// TODO: mock storage that returns error on put
}

func TestCopyAndRevertUtxos(t *testing.T) {
	db :=  storage.NewRamStorage()
	defer db.Close()

	coinbaseAddr := Address{"testaddress"}
	bc := CreateBlockchain(coinbaseAddr, db, nil)

	blk1 := GenerateUtxoMockBlockWithoutInputs()  // contains 2 UTXOs for address1
	blk2 := GenerateUtxoMockBlockWithInputs()  // contains tx that transfers address1's UTXOs to address2 with a change

	bc.AddBlockToTail(blk1)
	bc.AddBlockToTail(blk2)

	utxoIndex := LoadUTXOIndex(db)
	addr1UTXOs := utxoIndex.GetUTXOsByPubKey(address1Hash)
	addr2UTXOs := utxoIndex.GetUTXOsByPubKey(address2Hash)
	// Expect address1 to have 1 utxo of $4
	assert.Equal(t, 1, len(addr1UTXOs))
	assert.Equal(t, common.NewAmount(4),  addr1UTXOs[0].Value)

	// Expect address2 to have 2 utxos totaling $8
	assert.Equal(t, 2, len(addr2UTXOs))

	// Rollback to blk1, address1 has a $5 utxo and a $7 utxo, total $12, and address2 has nothing
	indexSnapshot, err := GetUTXOIndexAtBlockHash(db, bc, blk1.GetHash())
	if err !=nil {
		panic(err)
	}

	assert.Equal(t, 2, len(indexSnapshot[string(address1Hash)]))
	assert.Equal(t, common.NewAmount(5),  indexSnapshot[string(address1Hash)][0].Value)
	assert.Equal(t, common.NewAmount(7),  indexSnapshot[string(address1Hash)][1].Value)
	assert.Equal(t, 0,  len(indexSnapshot[string(address2Hash)]))
}

func TestFindUTXO(t *testing.T) {
	Txin := MockTxInputs()
	Txin = append(Txin, MockTxInputs()...)
	utxo1 := &UTXO{common.NewAmount(10),[]byte("addr1"),Txin[0].Txid,Txin[0].Vout}
	utxo2 := &UTXO{common.NewAmount(9),[]byte("addr1"),Txin[1].Txid,Txin[1].Vout}
	utxoIndex := make(UTXOIndex)
	utxoIndex["addr1"] = []*UTXO{utxo1, utxo2}

	assert.Equal(t, utxo1, utxoIndex.FindUTXO(Txin[0].Txid, Txin[0].Vout))
	assert.Equal(t, utxo2, utxoIndex.FindUTXO(Txin[1].Txid, Txin[1].Vout))
	assert.Nil(t, utxoIndex.FindUTXO(Txin[2].Txid, Txin[2].Vout))
	assert.Nil(t, utxoIndex.FindUTXO(Txin[3].Txid, Txin[3].Vout))
}
