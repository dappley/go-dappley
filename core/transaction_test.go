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

	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/util"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func GenerateFakeTxInputs() []TXInput {
	return []TXInput{
		{getAoB(2), 10, getAoB(2), getAoB(2)},
		{getAoB(2), 5, getAoB(2), getAoB(2)},
	}
}

func GenerateFakeTxOutputs() []TXOutput {
	return []TXOutput{
		{1, getAoB(2)},
		{2, getAoB(2)},
	}
}

func TestTrimmedCopy(t *testing.T) {
	var t1 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  2,
	}

	t2 := t1.TrimmedCopy()

	t3 := NewCoinbaseTX("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F", "")
	t4 := t3.TrimmedCopy()
	assert.Equal(t, t1.ID, t2.ID)
	assert.Equal(t, t1.Tip, t2.Tip)
	assert.Equal(t, t1.Vout, t2.Vout)
	for index, vin := range t2.Vin {
		assert.Nil(t, vin.Signature)
		assert.Nil(t, vin.PubKey)
		assert.Equal(t, t1.Vin[index].Txid, vin.Txid)
		assert.Equal(t, t1.Vin[index].Vout, vin.Vout)
	}

	assert.Equal(t, t3.ID, t4.ID)
	assert.Equal(t, t3.Tip, t4.Tip)
	assert.Equal(t, t3.Vout, t4.Vout)
	for index, vin := range t4.Vin {
		assert.Nil(t, vin.Signature)
		assert.Nil(t, vin.PubKey)
		assert.Equal(t, t3.Vin[index].Txid, vin.Txid)
		assert.Equal(t, t3.Vin[index].Vout, vin.Vout)
	}
}

func TestVerify(t *testing.T) {
	var prevTXs = map[string]Transaction{}

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
	prevTXs[string(t1.ID)] = t2
	prevTXs[string(t2.ID)] = t3
	prevTXs[string(t3.ID)] = t4

}

//test IsCoinBase and NewCoinbaseTX function
func TestIsCoinBase(t *testing.T) {
	var t1 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  2,
	}

	assert.False(t, t1.IsCoinbase())

	t2 := NewCoinbaseTX("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F", "")

	assert.True(t, t2.IsCoinbase())

}

func TestTransaction_Proto(t *testing.T) {
	t1 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  5,
	}

	pb := t1.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.Transaction{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	t2 := Transaction{}
	t2.FromProto(newpb)

	assert.Equal(t, t1, t2)
}

func TestTransaction_FindTxInUtxoPool(t *testing.T) {
	//prepare utxo pool
	Txin := MockTxInputs()
	Txin2 := MockTxInputs()
	utxo1 := UTXOutputStored{10,[]byte("addr1"),Txin[0].Txid,Txin[0].Vout}
	utxo2 := UTXOutputStored{9,[]byte("addr1"),Txin[1].Txid,Txin[1].Vout}
	utxo3 := UTXOutputStored{9,[]byte("addr1"),Txin2[0].Txid,Txin2[0].Vout}
	utxo4 := UTXOutputStored{9,[]byte("addr1"),Txin2[1].Txid,Txin2[1].Vout}
	utxoPool := utxoIndex{}
	utxoPool["addr1"] = []UTXOutputStored{utxo1, utxo2, utxo3, utxo4}

	tx := MockTransaction()
	assert.Nil(t, tx.FindAllTxinsInUtxoPool(utxoPool))
	tx.Vin = Txin
	assert.NotNil(t, tx.FindAllTxinsInUtxoPool(utxoPool))
}
