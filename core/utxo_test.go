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
	"github.com/dappley/go-dappley/common"
	"testing"
	"github.com/dappley/go-dappley/storage"
	"time"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"

	"fmt"
)

var 	bh1 = &BlockHeader{
	[]byte("hash"),
	nil,
	1,
	time.Now().Unix(),
	nil,
	0,
}

var 	bh2 = &BlockHeader{
	[]byte("hash1"),
	[]byte("hash"),
	1,
	time.Now().Unix(),
	nil,
	1,
}
func GenerateUtxoMockBlockWithoutInputs() *Block{


	t1 := MockUtxoTransactionWithoutInputs()
	return &Block{
		header:       bh1,
		transactions: []*Transaction{t1},
	}
}


func GenerateUtxoMockBlockWithInputs() *Block{

	t1 := MockUtxoTransactionWithInputs()
	return &Block{
		header:       bh2,
		transactions: []*Transaction{t1},
	}
}

func MockUtxoTransactionWithoutInputs() *Transaction{
	return &Transaction{
		ID:  []byte("tx1"),
		Vin:  []TXInput{},
		Vout: MockUtxoOutputsWithoutInputs(),
		Tip:  5,
	}
}

func MockUtxoTransactionWithInputs() *Transaction{
	return &Transaction{
		ID:   []byte("tx2"),
		Vin:  MockUtxoInputs(),
		Vout: MockUtxoOutputsWithInputs(),
		Tip:  5,
	}
}

func MockUtxoInputs() []TXInput {
	return []TXInput{
		{[]byte("tx1"),
			0,
			util.GenerateRandomAoB(2),
			[]byte("address1")},
		{[]byte("tx1"),
			1,
			util.GenerateRandomAoB(2),
			[]byte("address1")},
	}
}

func MockUtxoOutputsWithoutInputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(5), []byte("address1")},
		{common.NewAmount(7), []byte("address1")},
	}
}

func MockUtxoOutputsWithInputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(4), []byte("address1")},
		{common.NewAmount(5), []byte("address2")},
		{common.NewAmount(3), []byte("address2")},
	}
}

func TestAddSpendableOutputsAfterNewBlock(t *testing.T){
	db :=  storage.NewRamStorage()
	defer db.Close()
	blk := GenerateUtxoMockBlockWithoutInputs()

	blk.AddSpendableOutputsAfterNewBlock(UtxoMapKey, db)
	myUtxos := GetAddressUTXOs(UtxoMapKey, []byte("address1"), db)

	assert.Equal(t, common.NewAmount(5), myUtxos[0].Value )
	assert.Equal(t, common.NewAmount(7), myUtxos[1].Value )
}

func TestConsumeSpentOutputsAfterNewBlock(t *testing.T){
	db :=  storage.NewRamStorage()
	defer db.Close()

	blk1 := GenerateUtxoMockBlockWithoutInputs()

	blk1.AddSpendableOutputsAfterNewBlock(UtxoMapKey, db)
	//address 1 is given a $5 utxo and a $7 utxo, total $12

	blk2 := GenerateUtxoMockBlockWithInputs()
	//consume utxos first, not adding new utxos yet

	blk2.ConsumeSpendableOutputsAfterNewBlock(UtxoMapKey,db)
	//address1 gives address2 $8, $12 - $8 = $4 but address1 has no utxos left at this point new(change) utxo hasnt been added
	assert.Equal(t, 0, len( GetAddressUTXOs(UtxoMapKey, []byte("address1"), db)))

	//add utxos for above block accordingly;
	blk2.AddSpendableOutputsAfterNewBlock(UtxoMapKey,db)

	//expect address1 to have 1 utxo of $4
	assert.Equal(t, 1, len( GetAddressUTXOs(UtxoMapKey,[]byte("address1"), db)))
	assert.Equal(t, common.NewAmount(4),  GetAddressUTXOs(UtxoMapKey,[]byte("address1"), db)[0].Value)

	//expect address2 to have 2 utxos totaling $8
	assert.Equal(t, 2, len( GetAddressUTXOs(UtxoMapKey,[]byte("address2"), db)))
	sum := common.NewAmount(0)
	for _, utxo := range GetAddressUTXOs(UtxoMapKey, []byte("address2"),db) {
		sum = sum.Add(utxo.Value)
	}
	assert.Equal(t, common.NewAmount(8), sum)
}

func TestCopyAndRevertUtxosInRam(t *testing.T){

	db :=  storage.NewRamStorage()
	defer db.Close()
	addr1 := Address{"testaddress"}
	bc := CreateBlockchain(addr1, db, nil)

	blk1 := GenerateUtxoMockBlockWithoutInputs()
	blk2 := GenerateUtxoMockBlockWithInputs()

	bc.AddBlockToTail(blk1)
	bc.AddBlockToTail(blk2)
	//expect address1 to have 1 utxo of $4
	assert.Equal(t, 1, len(GetAddressUTXOs(UtxoMapKey,[]byte("address1"), db)))
	assert.Equal(t, common.NewAmount(4),  GetAddressUTXOs(UtxoMapKey,[]byte("address1"), db)[0].Value)

	//expect address2 to have 2 utxos totaling $8
	assert.Equal(t, 2, len( GetAddressUTXOs(UtxoMapKey,[]byte("address2"), db)))

	//rollback to block 1, address 1 has a $5 utxo and a $7 utxo, total $12, and addr2 has nothing
	deepCopy, err:= bc.GetUtxoStateAtBlockHash(db, blk1.GetHash())
	if err !=nil{
		panic(err)
	}

	assert.Equal(t, 2, len(deepCopy["address1"]))
	assert.Equal(t, common.NewAmount(5),  deepCopy["address1"][0].Value)
	assert.Equal(t, common.NewAmount(7),  deepCopy["address1"][1].Value)
	assert.Equal(t, 0,  len(deepCopy["address2"]))

}



func TestUtxoIndex_VerifyTransactionInput(t *testing.T) {
	Txin := MockTxInputs()
	Txin = append(Txin, MockTxInputs()...)
	utxo1 := UTXOutputStored{common.NewAmount(10),[]byte("addr1"),Txin[0].Txid,Txin[0].Vout}
	utxo2 := UTXOutputStored{common.NewAmount(9),[]byte("addr1"),Txin[1].Txid,Txin[1].Vout}
	utxoPool := utxoIndex{}
	utxoPool["addr1"] = []UTXOutputStored{utxo1, utxo2}

	assert.NotNil(t, utxoPool.FindUtxoByTxinput(Txin[0]))
	assert.NotNil(t, utxoPool.FindUtxoByTxinput(Txin[1]))
	assert.Nil(t, utxoPool.FindUtxoByTxinput(Txin[2]))
	assert.Nil(t, utxoPool.FindUtxoByTxinput(Txin[3]))
}

func TestUpdateUtxoIndexAfterNewBlock(t *testing.T){
	a := make(map[int]string)
	fmt.Println(a[1])
	assert.True(t, true)

}