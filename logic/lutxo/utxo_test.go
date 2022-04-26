// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package lutxo

import (
	"errors"
	"fmt"
	"github.com/dappley/go-dappley/core/block"
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/dappley/go-dappley/util"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/storage/mocks"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Padding Address to 32 Byte
var address1Bytes = []byte("address1000000000000000000000000")
var address2Bytes = []byte("address2000000000000000000000000")
var ta1 = account.NewTransactionAccountByPubKey(address1Bytes)
var ta2 = account.NewTransactionAccountByPubKey(address2Bytes)

func TestUTXOIndex_AddUTXO(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	txout := transactionbase.TXOutput{common.NewAmount(5), ta1.GetPubKeyHash(), ""}
	utxoIndex := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	utxoIndex.AddUTXO(txout, []byte{1}, 0)

	addr1UTXOs := utxoIndex.indexAdd[ta1.GetPubKeyHash().String()]
	assert.Equal(t, 1, addr1UTXOs.Size())
	assert.Equal(t, txout.Value, addr1UTXOs.GetAllUtxos()[0].Value)
	assert.Equal(t, []byte{1}, addr1UTXOs.GetAllUtxos()[0].Txid)
	assert.Equal(t, 0, addr1UTXOs.GetAllUtxos()[0].TxIndex)

	_, ok := utxoIndex.indexAdd["address2"]
	assert.Equal(t, false, ok)
}

func TestUTXOIndex_RemoveUTXO(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	utxoIndex := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	addr1UtxoTx := utxo.NewUTXOTx()
	addr1UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(5), ta1.GetPubKeyHash(), ""}, []byte{1}, 0, utxo.UtxoNormal, []byte{}, []byte{}})
	addr1UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(2), ta1.GetPubKeyHash(), ""}, []byte{1}, 1, utxo.UtxoNormal, []byte{}, []byte{}})
	addr1UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(2), ta1.GetPubKeyHash(), ""}, []byte{2}, 0, utxo.UtxoNormal, []byte{}, []byte{}})

	addr2UtxoTx := utxo.NewUTXOTx()
	addr2UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(4), ta2.GetPubKeyHash(), ""}, []byte{1}, 2, utxo.UtxoNormal, []byte{}, []byte{}})

	utxoIndex.indexAdd[ta1.GetPubKeyHash().String()] = &addr1UtxoTx
	utxoIndex.indexAdd[ta2.GetPubKeyHash().String()] = &addr2UtxoTx

	err := utxoIndex.removeUTXO(ta1.GetPubKeyHash(), []byte{1}, 0)

	assert.Nil(t, err)
	assert.Equal(t, 2, utxoIndex.indexAdd[ta1.GetPubKeyHash().String()].Size())
	assert.Equal(t, 1, utxoIndex.indexAdd[ta2.GetPubKeyHash().String()].Size())

	err = utxoIndex.removeUTXO(ta2.GetPubKeyHash(), []byte{2}, 1) // Does not exists

	assert.NotNil(t, err)
	assert.Equal(t, 2, utxoIndex.indexAdd[ta1.GetPubKeyHash().String()].Size())
	assert.Equal(t, 1, utxoIndex.indexAdd[ta2.GetPubKeyHash().String()].Size())
}

func TestUpdate_Failed(t *testing.T) {
	db := new(mocks.Storage)

	simulatedFailure := errors.New("simulated storage failure")
	db.On("Put", mock.Anything, mock.Anything).Return(simulatedFailure)
	db.On("Get", mock.Anything, mock.Anything).Return(nil, nil)

	blk := core.GenerateUtxoMockBlockWithoutInputs()
	utxoIndex := NewUTXOIndex(utxo.NewUTXOCache(db))
	utxoIndex.UpdateUtxos(blk.GetTransactions())
	err := utxoIndex.Save()
	assert.Equal(t, simulatedFailure, err)
	assert.Equal(t, 2, utxoIndex.GetAllUTXOsByPubKeyHash(ta1.GetPubKeyHash()).Size())
}

func TestFindUTXO(t *testing.T) {
	Txin := core.MockTxInputs()
	Txin = append(Txin, core.MockTxInputs()...)
	utxo1 := &utxo.UTXO{transactionbase.TXOutput{common.NewAmount(10), account.PubKeyHash([]byte("addr1")), ""}, Txin[0].Txid, Txin[0].Vout, utxo.UtxoNormal, []byte{}, []byte{}}
	utxo2 := &utxo.UTXO{transactionbase.TXOutput{common.NewAmount(9), account.PubKeyHash([]byte("addr1")), ""}, Txin[1].Txid, Txin[1].Vout, utxo.UtxoNormal, []byte{}, []byte{}}
	utxoTx1 := utxo.NewUTXOTxWithData(utxo1)
	utxoTx2 := utxo.NewUTXOTxWithData(utxo2)

	assert.Equal(t, utxo1, utxoTx1.GetUtxo(Txin[0].Txid, Txin[0].Vout))
	assert.Equal(t, utxo2, utxoTx2.GetUtxo(Txin[1].Txid, Txin[1].Vout))
	assert.Nil(t, utxoTx1.GetUtxo(Txin[2].Txid, Txin[2].Vout))
	assert.Nil(t, utxoTx2.GetUtxo(Txin[3].Txid, Txin[3].Vout))
}

func TestConcurrentUTXOindexReadWrite(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	var mu sync.Mutex
	var readOps uint64
	var addOps uint64
	var deleteOps uint64
	const concurrentUsers = 10
	exists := false

	// start 10 simultaneous goroutines to execute repeated
	// reads and writes, once per millisecond in
	// each goroutine.
	for r := 0; r < concurrentUsers; r++ {
		go func() {
			for {
				//perform a read
				index.GetAllUTXOsByPubKeyHash([]byte("asd"))
				atomic.AddUint64(&readOps, 1)
				//perform a write

				mu.Lock()
				tmpExists := exists
				mu.Unlock()
				if !tmpExists {
					index.AddUTXO(transactionbase.TXOutput{}, []byte("asd"), 65)
					atomic.AddUint64(&addOps, 1)
					mu.Lock()
					exists = true
					mu.Unlock()

				} else {
					index.removeUTXO([]byte("asd"), []byte("asd"), 65)
					atomic.AddUint64(&deleteOps, 1)
					mu.Lock()
					exists = false
					mu.Unlock()
				}

				time.Sleep(time.Millisecond * 1)
			}
		}()
	}

	time.Sleep(time.Second * 1)

	//if reports concurrent map writes, then test is broken, if passes, then test is correct
	assert.True(t, true)
}

func TestUTXOIndex_GetUpdatedUtxo(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	acc := account.NewAccount()
	txo := transactionbase.TXOutput{
		Value:      common.NewAmount(10),
		PubKeyHash: acc.GetPubKeyHash(),
		Contract:   "",
	}
	txid := []byte("test1")

	// utxo is not in indexAdd or cache
	result, err := index.GetUpdatedUtxo(acc.GetPubKeyHash(), txid, 0)
	assert.Nil(t, result)
	assert.Equal(t, errors.New("key is invalid"), err)
	// utxo is in indexAdd
	index.AddUTXO(txo, txid, 0)
	result, err = index.GetUpdatedUtxo(acc.GetPubKeyHash(), txid, 0)
	assert.Equal(t, utxo.NewUTXO(txo, txid, 0, utxo.UtxoNormal), result)
	assert.Nil(t, err)
	// utxo is in cache
	index.Save()
	result, err = index.GetUpdatedUtxo(acc.GetPubKeyHash(), txid, 0)
	assert.Equal(t, utxo.NewUTXO(txo, txid, 0, utxo.UtxoNormal), result)
	assert.Nil(t, err)
	// utxo is in indexRemove
	err = index.removeUTXO(acc.GetPubKeyHash(), txid, 0)
	assert.Nil(t, err)
	result, err = index.GetUpdatedUtxo(acc.GetPubKeyHash(), txid, 0)
	assert.Nil(t, result)
	assert.Equal(t, errors.New("the utxo already has been removed"), err)
	// utxo is not in indexAdd or cache
	index.Save()
	result, err = index.GetUpdatedUtxo(acc.GetPubKeyHash(), txid, 0)
	assert.Nil(t, result)
	assert.Equal(t, errors.New("key is invalid"), err)
}

func TestUTXOIndex_GetContractCreateUTXOByPubKeyHash(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	acc := account.NewContractTransactionAccount()

	// no utxo found
	result := index.GetContractCreateUTXOByPubKeyHash(acc.GetPubKeyHash())
	assert.Nil(t, result)

	txid := []byte("test")
	txo1 := transactionbase.TXOutput{
		Value:      common.NewAmount(10),
		PubKeyHash: acc.GetPubKeyHash(),
		Contract:   "contract",
	}
	txo2 := transactionbase.TXOutput{
		Value:      common.NewAmount(20),
		PubKeyHash: acc.GetPubKeyHash(),
		Contract:   "contract2",
	}
	index.AddUTXO(txo1, txid, 0)
	index.AddUTXO(txo2, txid, 1)
	err := index.Save()
	assert.Nil(t, err)

	result = index.GetContractCreateUTXOByPubKeyHash(acc.GetPubKeyHash())
	assert.Equal(t, txo1.Value, result.Value)
	assert.Equal(t, txo1.PubKeyHash, result.PubKeyHash)
	assert.Equal(t, txo1.Contract, result.Contract)
	assert.Equal(t, utxo.UtxoCreateContract, result.UtxoType)
}

func TestUTXOIndex_GetContractInvokeUTXOsByPubKeyHash(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	acc := account.NewContractTransactionAccount()
	txo := transactionbase.TXOutput{
		Value:      common.NewAmount(10),
		PubKeyHash: acc.GetPubKeyHash(),
		Contract:   "contract",
	}
	txo1 := transactionbase.TXOutput{
		Value:      common.NewAmount(20),
		PubKeyHash: acc.GetPubKeyHash(),
		Contract:   "contract",
	}
	txo2 := transactionbase.TXOutput{
		Value:      common.NewAmount(30),
		PubKeyHash: acc.GetPubKeyHash(),
		Contract:   "contract",
	}
	txo3 := transactionbase.TXOutput{
		Value:      common.NewAmount(40),
		PubKeyHash: acc.GetPubKeyHash(),
		Contract:   "contract",
	}

	txid := []byte("test")
	index.AddUTXO(txo, txid, 0)
	// first one added is of type UtxoCreateContract so it shouldn't be returned
	result := index.GetContractInvokeUTXOsByPubKeyHash(acc.GetPubKeyHash())
	assert.Nil(t, result)

	index.AddUTXO(txo1, txid, 1)
	index.AddUTXO(txo2, txid, 2)
	index.AddUTXO(txo3, txid, 3)

	expected := []*utxo.UTXO{
		utxo.NewUTXO(txo1, txid, 1, utxo.UtxoInvokeContract),
		utxo.NewUTXO(txo2, txid, 2, utxo.UtxoInvokeContract),
		utxo.NewUTXO(txo3, txid, 3, utxo.UtxoInvokeContract),
	}

	result = index.GetContractInvokeUTXOsByPubKeyHash(acc.GetPubKeyHash())
	assert.Equal(t, expected, result)
}

func TestUTXOIndex_GetUTXOsAccordingToAmount(t *testing.T) {
	contractAccount := account.NewContractTransactionAccount()
	contractPkh := contractAccount.GetPubKeyHash()
	//preapre 3 utxos in the utxo index
	TXOutputs := []transactionbase.TXOutput{
		{common.NewAmount(3), ta1.GetPubKeyHash(), ""},
		{common.NewAmount(4), ta2.GetPubKeyHash(), ""},
		{common.NewAmount(5), ta2.GetPubKeyHash(), ""},
		{common.NewAmount(2), contractPkh, "helloworld!"},
		{common.NewAmount(4), contractPkh, ""},
	}
	db := storage.NewRamStorage()
	defer db.Close()
	index := NewUTXOIndex(utxo.NewUTXOCache(db))
	for i, TXOutput := range TXOutputs {
		index.AddUTXO(TXOutput, []byte("01"), i)
	}

	//start the test
	tests := []struct {
		name   string
		amount *common.Amount
		pubKey []byte
		err    error
	}{
		{"enoughUtxo",
			common.NewAmount(3),
			[]byte(ta2.GetPubKeyHash()),
			nil},

		{"notEnoughUtxo",
			common.NewAmount(4),
			[]byte(ta1.GetPubKeyHash()),
			transaction.ErrInsufficientFund},

		{"justEnoughUtxo",
			common.NewAmount(9),
			[]byte(ta2.GetPubKeyHash()),
			nil},
		{"notEnoughUtxo2",
			common.NewAmount(10),
			[]byte(ta2.GetPubKeyHash()),
			transaction.ErrInsufficientFund},
		{"smartContractUtxo",
			common.NewAmount(3),
			[]byte(contractPkh),
			nil},
		{"smartContractUtxoInsufficient",
			common.NewAmount(5),
			[]byte(contractPkh),
			transaction.ErrInsufficientFund},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utxos, err := index.GetUTXOsAccordingToAmount(tt.pubKey, tt.amount)
			assert.Equal(t, tt.err, err)
			if err != nil {
				return
			}
			sum := common.NewAmount(0)
			for _, utxo := range utxos {
				sum = sum.Add(utxo.Value)
			}
			assert.True(t, sum.Cmp(tt.amount) >= 0)
		})
	}
}

func TestUTXOIndex_getUTXOsFromCacheUTXO(t *testing.T) {
	acc := account.NewAccount()
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	outputContract := transactionbase.TXOutput{Value: common.NewAmount(1), PubKeyHash: acc.GetPubKeyHash(), Contract: "contract"}
	output1 := transactionbase.TXOutput{Value: common.NewAmount(20), PubKeyHash: acc.GetPubKeyHash(), Contract: ""}
	output2 := transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""}
	output3 := transactionbase.TXOutput{Value: common.NewAmount(5), PubKeyHash: acc.GetPubKeyHash(), Contract: ""}

	utxoContract := utxo.NewUTXO(outputContract, []byte{0x87}, 0, utxo.UtxoCreateContract)
	utxo1 := utxo.NewUTXO(output1, []byte{0x88}, 0, utxo.UtxoNormal)
	utxo2 := utxo.NewUTXO(output2, []byte{0x88}, 1, utxo.UtxoNormal)
	utxo3 := utxo.NewUTXO(output3, []byte{0x88}, 2, utxo.UtxoNormal)

	indexAddUtxoTx := utxo.NewUTXOTx()
	indexAddUtxoTx.PutUtxo(utxoContract)
	indexAddUtxoTx.PutUtxo(utxo1)
	indexAddUtxoTx.PutUtxo(utxo2)
	indexAddUtxoTx.PutUtxo(utxo3)

	indexRemoveUtxoTx := utxo.NewUTXOTx()
	indexRemoveUtxoTx.PutUtxo(utxo1)
	index.indexAdd[acc.GetPubKeyHash().String()] = &indexAddUtxoTx
	index.indexRemove[acc.GetPubKeyHash().String()] = &indexRemoveUtxoTx

	// amount only requires one utxo
	remove, utxos, amount, err := index.getUTXOsFromCacheUTXO(acc.GetPubKeyHash(), common.NewAmount(1))
	assert.Equal(t, &indexRemoveUtxoTx, remove)
	assert.Equal(t, 1, len(utxos))
	// map access is random so either utxo2 or utxo3 is valid
	if common.NewAmount(10).Cmp(amount) == 0 {
		assert.Equal(t, []*utxo.UTXO{utxo2}, utxos)
	} else {
		assert.Equal(t, []*utxo.UTXO{utxo3}, utxos)
		assert.Equal(t, common.NewAmount(5), amount)
	}
	assert.Nil(t, err)

	// amount requires all utxos
	remove, utxos, amount, err = index.getUTXOsFromCacheUTXO(acc.GetPubKeyHash(), common.NewAmount(200))
	assert.Equal(t, &indexRemoveUtxoTx, remove)
	assert.ElementsMatch(t, []*utxo.UTXO{utxo2, utxo3}, utxos)
	assert.Equal(t, common.NewAmount(15), amount)
	assert.Nil(t, err)
}

func TestUTXOIndex_UpdateUtxo(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	acc1 := account.NewAccount()
	acc2 := account.NewAccount()

	// current transaction
	tx := &transaction.Transaction{
		ID: []byte("current"),
		Vin: []transactionbase.TXInput{
			{
				Txid:      []byte("previous"),
				Vout:      0,
				Signature: nil,
				PubKey:    []byte{},
			},
		},
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(10),
				PubKeyHash: acc2.GetPubKeyHash(),
				Contract:   "",
			},
		},
		Tip:        common.NewAmount(2),
		GasLimit:   common.NewAmount(30000),
		GasPrice:   common.NewAmount(1),
		CreateTime: 0,
		Type:       transaction.TxTypeNormal,
	}

	// tx has invalid Vin PubKey
	ok := index.UpdateUtxo(tx)
	assert.False(t, ok)
	tx.Vin[0].PubKey = acc1.GetKeyPair().GetPublicKey()

	// utxo to remove does not exist
	ok = index.UpdateUtxo(tx)
	assert.False(t, ok)

	// add previous utxo for tx.Vin
	prevTxOut := transactionbase.TXOutput{
		Value:      common.NewAmount(25),
		PubKeyHash: acc1.GetPubKeyHash(),
		Contract:   "",
	}
	index.AddUTXO(prevTxOut, []byte("previous"), 0)

	ok = index.UpdateUtxo(tx)
	assert.True(t, ok)
}

func TestUTXOIndex_DeepCopy(t *testing.T) {
	utxoIndex := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	utxoCopy := utxoIndex.DeepCopy()
	assert.Equal(t, 0, len(utxoIndex.indexAdd))
	assert.Equal(t, 0, len(utxoCopy.indexAdd))

	addr1UtxoTx := utxo.NewUTXOTx()
	utxoIndex.indexAdd[string(ta1.GetPubKeyHash())] = &addr1UtxoTx
	assert.Equal(t, 1, len(utxoIndex.indexAdd))
	assert.Equal(t, 0, len(utxoCopy.indexAdd))

	copyUtxoTx := utxo.NewUTXOTxWithData(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, utxo.UtxoNormal, []byte{}, []byte{}})
	utxoCopy.indexAdd[string(ta1.GetPubKeyHash())] = &copyUtxoTx
	assert.Equal(t, 1, len(utxoIndex.indexAdd))
	assert.Equal(t, 1, len(utxoCopy.indexAdd))
	assert.Equal(t, 0, utxoIndex.indexAdd[string(ta1.GetPubKeyHash())].Size())
	assert.Equal(t, 1, utxoCopy.indexAdd[string(ta1.GetPubKeyHash())].Size())

	copyUtxoTx1 := utxo.NewUTXOTx()
	copyUtxoTx1.PutUtxo(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, utxo.UtxoNormal, []byte{}, []byte{}})
	copyUtxoTx1.PutUtxo(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[1], []byte{}, 1, utxo.UtxoNormal, []byte{}, []byte{}})
	utxoCopy.indexAdd["1"] = &copyUtxoTx1

	utxoCopy2 := utxoCopy.DeepCopy()
	copy2UtxoTx1 := utxo.NewUTXOTx()
	copy2UtxoTx1.PutUtxo(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, utxo.UtxoNormal, []byte{}, []byte{}})
	utxoCopy2.indexAdd["1"] = &copy2UtxoTx1
	assert.Equal(t, 2, len(utxoCopy.indexAdd))
	assert.Equal(t, 2, len(utxoCopy2.indexAdd))
	assert.Equal(t, 2, utxoCopy.indexAdd["1"].Size())
	assert.Equal(t, 1, utxoCopy2.indexAdd["1"].Size())
	assert.Equal(t, 1, len(utxoIndex.indexAdd))

	assert.EqualValues(t, utxoCopy.indexAdd[ta1.GetPubKeyHash().String()], utxoCopy2.indexAdd[ta1.GetPubKeyHash().String()])
}

func TestUTXOIndex_Save(t *testing.T) {

	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var ta1 = account.NewAccountByPrivateKey(prikey1)
	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var ta2 = account.NewAccountByPrivateKey(prikey2)
	var prikey3 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa73"
	var ta3 = account.NewAccountByPrivateKey(prikey3)
	var prikey4 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa74"
	var ta4 = account.NewAccountByPrivateKey(prikey4)
	var prikey5 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa75"
	var ta5 = account.NewAccountByPrivateKey(prikey5)

	var dependentTx1 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{util.GenerateRandomAoB(1), 1, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta1.GetPubKeyHash(), ""},
			{common.NewAmount(10), ta2.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(3),
	}
	dependentTx1.ID = dependentTx1.Hash()

	utxoPk10 := &utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}
	utxoPk11 := &utxo.UTXO{dependentTx1.Vout[1], dependentTx1.ID, 1, utxo.UtxoNormal, []byte{}, []byte{}}

	var dependentTx2 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx1.ID, 1, nil, ta2.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta3.GetPubKeyHash(), ""},
			{common.NewAmount(3), ta4.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(2),
	}
	dependentTx2.ID = dependentTx2.Hash()
	//ta1 5,ta2 0,ta3 5,ta4 5
	utxoPk20 := &utxo.UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}
	utxoPk21 := &utxo.UTXO{dependentTx2.Vout[1], dependentTx2.ID, 1, utxo.UtxoNormal, []byte{}, []byte{}}

	var dependentTx3 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx2.ID, 0, nil, ta3.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(1), ta4.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx3.ID = dependentTx3.Hash()
	//ta1 5,ta2 0,ta3 0,ta4 5+1
	utxoPk30 := &utxo.UTXO{dependentTx3.Vout[0], dependentTx3.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}

	var dependentTx4 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx2.ID, 1, nil, ta4.GetKeyPair().GetPublicKey()},
			{dependentTx3.ID, 0, nil, ta4.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(3), ta1.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(1),
	}
	dependentTx4.ID = dependentTx4.Hash()
	//ta1 5+3,ta2 0,ta3 0,ta4 6-3-1
	utxoPk40 := &utxo.UTXO{dependentTx4.Vout[0], dependentTx4.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}

	var dependentTx5 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx1.ID, 0, nil, ta1.GetKeyPair().GetPublicKey()},
			{dependentTx4.ID, 0, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(4), ta5.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx5.ID = dependentTx5.Hash()
	//ta1 8-4-4,ta2 0,ta3 0,ta4 2
	//ta1 0,ta2 0,ta3 0,ta4 2,ta5 4
	utxoPk50 := &utxo.UTXO{dependentTx5.Vout[0], dependentTx5.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}

	db := storage.NewRamStorage()
	defer db.Close()
	utxoIndex := NewUTXOIndex(utxo.NewUTXOCache(db))

	utxoTx1 := utxo.NewUTXOTx() //ta1
	utxoTx1.PutUtxo(utxoPk10)
	utxoTx1.PutUtxo(utxoPk40)

	utxoTx2 := utxo.NewUTXOTx() //ta2
	utxoTx2.PutUtxo(utxoPk11)

	utxoTx3 := utxo.NewUTXOTx() //ta3
	utxoTx3.PutUtxo(utxoPk20)

	utxoTx4 := utxo.NewUTXOTx() //ta4
	utxoTx4.PutUtxo(utxoPk21)
	utxoTx4.PutUtxo(utxoPk30)

	utxoTx5 := utxo.NewUTXOTx() //ta5
	utxoTx5.PutUtxo(utxoPk50)

	utxoIndex.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx1,
		ta2.GetPubKeyHash().String(): &utxoTx2,
		ta3.GetPubKeyHash().String(): &utxoTx3,
		ta4.GetPubKeyHash().String(): &utxoTx4,
		ta5.GetPubKeyHash().String(): &utxoTx5,
	})

	utxoIndex.SetindexRemove(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx1,
		ta2.GetPubKeyHash().String(): &utxoTx2,
		ta3.GetPubKeyHash().String(): &utxoTx3,
		ta4.GetPubKeyHash().String(): &utxoTx4,
	})

	//test add and remove utxo
	err := utxoIndex.Save()
	assert.Nil(t, err)
	assert.Equal(t, false, utxoIndex.IsLastUtxoKeyExist(ta1.GetPubKeyHash()))
	assert.Equal(t, false, utxoIndex.IsLastUtxoKeyExist(ta2.GetPubKeyHash()))
	assert.Equal(t, false, utxoIndex.IsLastUtxoKeyExist(ta3.GetPubKeyHash()))
	assert.Equal(t, false, utxoIndex.IsLastUtxoKeyExist(ta4.GetPubKeyHash()))
	assert.Equal(t, true, utxoIndex.IsLastUtxoKeyExist(ta5.GetPubKeyHash()))

	//remove utxo which not in db
	utxoIndex.SetindexRemove(map[string]*utxo.UTXOTx{
		ta4.GetPubKeyHash().String(): &utxoTx4,
	})
	err = utxoIndex.Save()
	assert.Equal(t, errors.New("key is invalid"), err)

	//add a utxo which is same as last utxo
	utxoIndex.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta5.GetPubKeyHash().String(): &utxoTx5,
	})
	err = utxoIndex.Save()
	assert.Equal(t, errors.New("add utxo failed: the utxo is same as the last utxo"), err)

	utxoIndex2 := NewUTXOIndex(utxo.NewUTXOCache(db))
	utxoTx10 := utxo.NewUTXOTx() //ta1
	utxoTx10.PutUtxo(utxoPk10)
	utxoIndex2.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx10, //first time add utxoPk10
	})
	err = utxoIndex2.Save()
	assert.Nil(t, err)

	utxoTx40 := utxo.NewUTXOTx() //ta1
	utxoTx40.PutUtxo(utxoPk40)
	utxoIndex2.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx40, //add utxoPk40
	})
	err = utxoIndex2.Save()
	assert.Nil(t, err)

	utxoTx1Add := utxo.NewUTXOTx()
	utxoTx1Add.PutUtxo(utxoPk10) //second time add utxoPk10
	utxoIndex2.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx1Add,
	})
	utxoTx1Remove := utxo.NewUTXOTx()
	utxoTx1Remove.PutUtxo(utxoPk40) //delete utxoPK40 will connect two utxoPk10 together, which should be captured.
	utxoIndex2.SetindexRemove(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx1Remove,
	})
	err = utxoIndex2.Save()
	assert.Equal(t, errors.New("remove utxo error: find duplicate utxo in db"), err)

	//The following print outs are normal, because the utxoInfo has not been created
	// until the first pubkey's utxo is stored.
	//time="2021-01-27T16:37:06-08:00" level=warning msg="utxoInfo not found in db"
	//time="2021-01-27T16:37:06-08:00" level=warning msg="getLastUTXOKey error:key is invalid"
	//time="2021-01-27T16:37:06-08:00" level=warning msg="utxoInfo not found in db"
	//time="2021-01-27T16:37:06-08:00" level=warning msg="key is invalid"

}

func TestUTXOIndex_IsIndexAddExist(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	acc := account.NewContractTransactionAccount()
	assert.False(t, index.IsIndexAddExist(acc.GetPubKeyHash()))

	index.SetIndexAdd(map[string]*utxo.UTXOTx{
		acc.GetPubKeyHash().String(): &utxo.UTXOTx{},
	})
	assert.True(t, index.IsIndexAddExist(acc.GetPubKeyHash()))

	index.SetIndexAdd(make(map[string]*utxo.UTXOTx))
	assert.False(t, index.IsIndexAddExist(acc.GetPubKeyHash()))
}

func TestUTXOIndex_AddAndRmoveUTXO(t *testing.T) {
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var ta1 = account.NewAccountByPrivateKey(prikey1)

	//test add utxo1
	var dependentTx1 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{util.GenerateRandomAoB(1), 1, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta1.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(3),
	}
	dependentTx1.ID = dependentTx1.Hash()
	utxo1 := &utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}

	db := storage.NewRamStorage()
	defer db.Close()
	utxoCache := utxo.NewUTXOCache(db)
	utxoIndex := NewUTXOIndex(utxoCache)
	utxoTx1 := utxo.NewUTXOTx()
	utxoTx1.PutUtxo(utxo1)

	SetIndexAddAndSave := func(utxoTx utxo.UTXOTx) error {
		utxoIndex.SetIndexAdd(map[string]*utxo.UTXOTx{
			ta1.GetPubKeyHash().String(): &utxoTx,
		})
		return utxoIndex.Save()
	}

	err := SetIndexAddAndSave(utxoTx1)
	assert.Nil(t, err)

	utxoNew := &utxo.UTXO{}
	getUTXOValue := func(utxoKey string) (*common.Amount, []byte, []byte, error) {
		rawBytes, err := db.Get(util.Str2bytes(utxoKey))
		if err != nil {
			return nil, nil, nil, err
		}
		utxoPb := &utxopb.Utxo{}
		err = proto.Unmarshal(rawBytes, utxoPb)
		if err != nil {
			return nil, nil, nil, err
		}
		utxoNew.FromProto(utxoPb)
		return utxoNew.Value, utxoNew.PrevUtxoKey, utxoNew.NextUtxoKey, nil
	}

	//chain: utxo1
	utxoValue, prevKey, nextKey, err := getUTXOValue(utxo1.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(5), utxoValue)
	assert.Equal(t, []byte(nil), prevKey)
	assert.Equal(t, []byte(nil), nextKey)

	//test add 2 utxos: utxo20 and utxo21
	var dependentTx2 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{util.GenerateRandomAoB(1), 1, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(4), ta1.GetPubKeyHash(), ""},
			{common.NewAmount(6), ta1.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(3),
	}

	dependentTx2.ID = dependentTx2.Hash()
	utxo20 := &utxo.UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}
	utxo21 := &utxo.UTXO{dependentTx2.Vout[1], dependentTx2.ID, 1, utxo.UtxoNormal, []byte{}, []byte{}}

	utxoTx20 := utxo.NewUTXOTx()
	utxoTx20.PutUtxo(utxo20)
	err = SetIndexAddAndSave(utxoTx20)
	assert.Nil(t, err)

	//UTXOTx is map which may cause disorder, so save utxo one by one
	utxoTx21 := utxo.NewUTXOTx()
	utxoTx21.PutUtxo(utxo21)
	err = SetIndexAddAndSave(utxoTx21)
	assert.Nil(t, err)

	//chain: utxo21-utxo20-utxo1
	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo20.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(4), utxoValue)
	assert.Equal(t, util.Str2bytes(utxo21.GetUTXOKey()), prevKey)
	assert.Equal(t, util.Str2bytes(utxo1.GetUTXOKey()), nextKey)

	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo21.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(6), utxoValue)
	assert.Equal(t, []byte(nil), prevKey)
	assert.Equal(t, util.Str2bytes(utxo20.GetUTXOKey()), nextKey)

	//test delete tail utxo:utxo1
	SetIndexRemoveAndSave := func(utxoTx utxo.UTXOTx) error {
		utxoIndex.SetindexRemove(map[string]*utxo.UTXOTx{
			ta1.GetPubKeyHash().String(): &utxoTx,
		})
		return utxoIndex.Save()
	}
	err = SetIndexRemoveAndSave(utxoTx1)
	assert.Nil(t, err)

	//chain: utxo21-utxo20
	utxoValue, _, _, err = getUTXOValue(utxo1.GetUTXOKey())
	assert.Equal(t, errors.New("key is invalid"), err)

	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo20.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(4), utxoValue)
	assert.Equal(t, util.Str2bytes(utxo21.GetUTXOKey()), prevKey)
	assert.Equal(t, []byte(nil), nextKey)

	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo21.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(6), utxoValue)
	assert.Equal(t, []byte(nil), prevKey)
	assert.Equal(t, util.Str2bytes(utxo20.GetUTXOKey()), nextKey)

	//add a utxo：utxo3
	var dependentTx3 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{util.GenerateRandomAoB(1), 1, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(8), ta1.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(3),
	}
	dependentTx3.ID = dependentTx3.Hash()
	utxo3 := &utxo.UTXO{dependentTx3.Vout[0], dependentTx3.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}
	utxoTx30 := utxo.NewUTXOTx()
	utxoTx30.PutUtxo(utxo3)
	err = SetIndexAddAndSave(utxoTx30)
	assert.Nil(t, err)

	//chain: utxo3-utxo21-utxo20
	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo3.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(8), utxoValue)
	assert.Equal(t, []byte(nil), prevKey)
	assert.Equal(t, util.Str2bytes(utxo21.GetUTXOKey()), nextKey)

	//test delete middle utxo: utxo21
	utxoTx3 := utxo.NewUTXOTx()
	utxoTx3.PutUtxo(utxo21)
	err = SetIndexRemoveAndSave(utxoTx3)
	assert.Nil(t, err)

	//chain: utxo3-utxo20
	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo21.GetUTXOKey())
	assert.Equal(t, errors.New("key is invalid"), err)

	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo3.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(8), utxoValue)
	assert.Equal(t, []byte(nil), prevKey)
	assert.Equal(t, util.Str2bytes(utxo20.GetUTXOKey()), nextKey)

	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo20.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(4), utxoValue)
	assert.Equal(t, util.Str2bytes(utxo3.GetUTXOKey()), prevKey)
	assert.Equal(t, []byte(nil), nextKey)

	//	test delete first utxo in the chain:utxo3
	err = SetIndexRemoveAndSave(utxoTx30)
	assert.Nil(t, err)

	//chain: utxo20
	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo1.GetUTXOKey())
	assert.Equal(t, errors.New("key is invalid"), err)

	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo20.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(4), utxoValue)
	assert.Equal(t, []byte(nil), prevKey)
	assert.Equal(t, []byte(nil), nextKey)

	//test delete only one utxo left: utxo20
	err = SetIndexRemoveAndSave(utxoTx20)
	assert.Nil(t, err)
	//chain:
	utxoValue, prevKey, nextKey, err = getUTXOValue(utxo20.GetUTXOKey())
	assert.Equal(t, errors.New("key is invalid"), err)
}

func TestUTXOIndex_SelfCheckingUTXO(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	var txos []transactionbase.TXOutput
	var utxoTxs []utxo.UTXOTx
	for i := 0; i < 4; i++ {
		pkhString := fmt.Sprintf("hash%d", i)
		txo := transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte(pkhString),
			Contract:   "",
		}
		txos = append(txos, txo)

		txid := fmt.Sprintf("tx%d", i)
		utxoToAdd := utxo.NewUTXO(txo, []byte(txid), 0, utxo.UtxoNormal)
		utxoTx := utxo.NewUTXOTx()
		utxoTx.PutUtxo(utxoToAdd)
		utxoTxs = append(utxoTxs, utxoTx)
	}
	index.indexAdd[txos[0].PubKeyHash.String()] = &utxoTxs[0]
	index.indexAdd[txos[1].PubKeyHash.String()] = &utxoTxs[1]
	index.cache.AddUtxos(&utxoTxs[1], txos[1].PubKeyHash.String())
	index.cache.AddUtxos(&utxoTxs[2], txos[2].PubKeyHash.String())
	index.indexRemove[txos[2].PubKeyHash.String()] = &utxoTxs[2]
	index.indexRemove[txos[3].PubKeyHash.String()] = &utxoTxs[3]

	index.SelfCheckingUTXO()
	// expected outcome:
	// utxo 0 not removed
	// utxo 1 is removed from indexAdd (already in cache)
	// utxo 2 not removed
	// utxo 3 is removed from indexRemove (not in cache)
	tests := []struct {
		name        string
		pkh         account.PubKeyHash
		indexAdd    bool
		indexRemove bool
	}{
		{
			name:        "not removed from indexAdd",
			pkh:         txos[0].PubKeyHash,
			indexAdd:    true,
			indexRemove: false,
		},
		{
			name:        "removed from indexAdd",
			pkh:         txos[1].PubKeyHash,
			indexAdd:    false,
			indexRemove: false,
		},
		{
			name:        "not removed from indexRemove",
			pkh:         txos[2].PubKeyHash,
			indexAdd:    false,
			indexRemove: true,
		},
		{
			name:        "removed from indexRemove",
			pkh:         txos[3].PubKeyHash,
			indexAdd:    false,
			indexRemove: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := index.indexAdd[tt.pkh.String()]
			assert.Equal(t, tt.indexAdd, ok)

			_, ok = index.indexRemove[tt.pkh.String()]
			assert.Equal(t, tt.indexRemove, ok)
		})
	}
}

func TestFindVinUtxosInUtxoPool(t *testing.T) {
	acc := account.NewAccount()
	contractAcc := account.NewContractTransactionAccount()
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	tx := &transaction.Transaction{
		ID: []byte{0x89},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0x88}, Vout: 0, Signature: nil, PubKey: contractAcc.GetPubKeyHash()},
			{Txid: []byte{0x88}, Vout: 1, Signature: nil, PubKey: []byte("invalid")},
			{Txid: []byte{0x88}, Vout: 2, Signature: nil, PubKey: acc.GetKeyPair().GetPublicKey()},
		},
		Vout:       []transactionbase.TXOutput{},
		Tip:        common.NewAmount(1),
		GasLimit:   common.NewAmount(30000),
		GasPrice:   common.NewAmount(2),
		CreateTime: 0,
		Type:       transaction.TxTypeCoinbase,
	}

	// reject coinbase tx
	utxos, err := FindVinUtxosInUtxoPool(index, tx)
	assert.Nil(t, utxos)
	assert.Equal(t, transaction.ErrTXInputNotFound, err)
	tx.Type = transaction.TxTypeNormal

	output1 := transactionbase.TXOutput{Value: common.NewAmount(20), PubKeyHash: contractAcc.GetPubKeyHash(), Contract: ""}
	output2 := transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""}
	index.AddUTXO(output1, []byte{0x88}, 0)
	index.AddUTXO(output2, []byte{0x88}, 1)

	// vin has invalid pubkey
	utxos, err = FindVinUtxosInUtxoPool(index, tx)
	assert.Nil(t, utxos)
	assert.Equal(t, transaction.ErrNewUserPubKeyHash, err)
	tx.Vin[1].PubKey = acc.GetKeyPair().GetPublicKey()

	// index does not contain all vins in tx
	utxos, err = FindVinUtxosInUtxoPool(index, tx)
	assert.Nil(t, utxos)
	assert.Equal(t, transaction.ErrTXInputNotFound, err)

	// all vins are in index
	output3 := transactionbase.TXOutput{Value: common.NewAmount(5), PubKeyHash: acc.GetPubKeyHash(), Contract: ""}
	index.AddUTXO(output3, []byte{0x88}, 2)
	expectedUtxos := []*utxo.UTXO{
		index.indexAdd[contractAcc.GetPubKeyHash().String()].GetUtxo([]byte{0x88}, 0),
		index.indexAdd[acc.GetPubKeyHash().String()].GetUtxo([]byte{0x88}, 1),
		index.indexAdd[acc.GetPubKeyHash().String()].GetUtxo([]byte{0x88}, 2),
	}
	utxos, err = FindVinUtxosInUtxoPool(index, tx)
	assert.ElementsMatch(t, expectedUtxos, utxos)
	assert.Nil(t, err)
}

func TestGetTXOutputSpent(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	txin := transactionbase.TXInput{
		Txid:      []byte("test"),
		Vout:      0,
		Signature: nil,
		PubKey:    nil,
	}
	txout, vout, err := getTXOutputSpent(txin, db)
	assert.NotNil(t, err)

	tx := transaction.Transaction{
		ID: []byte("test"),
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(10),
				PubKeyHash: nil,
				Contract:   "contract1",
			},
			{
				Value:      common.NewAmount(20),
				PubKeyHash: nil,
				Contract:   "contract2",
			},
		},
	}
	err = transaction.PutTxJournal(tx, db)
	assert.Nil(t, err)

	txout, vout, err = getTXOutputSpent(txin, db)
	assert.Nil(t, err)
	assert.Equal(t, txin.Vout, vout)
	assert.Equal(t, tx.Vout[0], txout)

	txin.Vout = 1
	txout, vout, err = getTXOutputSpent(txin, db)
	assert.Nil(t, err)
	assert.Equal(t, txin.Vout, vout)
	assert.Equal(t, tx.Vout[1], txout)
}

func TestUTXOIndex_unspendVinsInTx(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	db := storage.NewRamStorage()
	defer db.Close()

	acc := account.NewAccount()

	// voutTx contains the vouts that will be inserted into the db, for the next tx to unspend
	voutTx := transaction.Transaction{
		ID: []byte("test"),
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(10),
				PubKeyHash: acc.GetPubKeyHash(),
				Contract:   "contract1",
			},
			{
				Value:      common.NewAmount(20),
				PubKeyHash: acc.GetPubKeyHash(),
				Contract:   "contract2",
			},
		},
	}
	err := transaction.PutTxJournal(voutTx, db)
	assert.Nil(t, err)

	// tx1 for unspending one vin
	tx1 := &transaction.Transaction{
		ID: []byte("test2"),
		Vin: []transactionbase.TXInput{
			{
				Txid:      []byte("test"),
				Vout:      0,
				Signature: nil,
				PubKey:    nil,
			},
		},
	}
	err = index.unspendVinsInTx(tx1, db)
	assert.Nil(t, err)

	utxoTx, ok := index.indexAdd[acc.GetPubKeyHash().String()]
	assert.True(t, ok)
	assert.NotNil(t, utxoTx)
	assert.Equal(t, 1, utxoTx.Size())
	utxo1 := utxoTx.GetUtxo(voutTx.ID, 0)
	assert.Equal(t, voutTx.Vout[0].Value, utxo1.Value)
	assert.Equal(t, voutTx.Vout[0].Contract, utxo1.Contract)

	index.SetIndexAdd(make(map[string]*utxo.UTXOTx))

	// tx2 for unspending multiple vins from one transaction
	tx2 := &transaction.Transaction{
		ID: []byte("test2"),
		Vin: []transactionbase.TXInput{
			{
				Txid:      []byte("test"),
				Vout:      1,
				Signature: nil,
				PubKey:    nil,
			},
			{
				Txid:      []byte("test"),
				Vout:      0,
				Signature: nil,
				PubKey:    nil,
			},
		},
	}
	err = index.unspendVinsInTx(tx2, db)
	assert.Nil(t, err)
	utxoTx, ok = index.indexAdd[acc.GetPubKeyHash().String()]
	assert.True(t, ok)
	assert.NotNil(t, utxoTx)
	assert.Equal(t, 2, utxoTx.Size())

	utxo1 = utxoTx.GetUtxo(voutTx.ID, 0)
	assert.Equal(t, voutTx.Vout[0].Value, utxo1.Value)
	assert.Equal(t, voutTx.Vout[0].Contract, utxo1.Contract)
	utxo2 := utxoTx.GetUtxo(voutTx.ID, 1)
	assert.Equal(t, voutTx.Vout[1].Value, utxo2.Value)
	assert.Equal(t, voutTx.Vout[1].Contract, utxo2.Contract)
}

func TestUTXOIndex_excludeVoutsInTx(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	db := storage.NewRamStorage()
	defer db.Close()

	acc := account.NewAccount()

	tx := &transaction.Transaction{
		ID: []byte("test"),
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(10),
				PubKeyHash: acc.GetPubKeyHash(),
				Contract:   "contract1",
			},
			{
				Value:      common.NewAmount(20),
				PubKeyHash: acc.GetPubKeyHash(),
				Contract:   "contract2",
			},
		},
	}

	for i := 0; i < len(tx.Vout); i++ {
		index.AddUTXO(tx.Vout[i], tx.ID, i)
	}
	err := index.Save()
	assert.Nil(t, err)
	assert.Nil(t, index.indexRemove[acc.GetPubKeyHash().String()])

	err = index.excludeVoutsInTx(tx, nil)
	assert.Nil(t, err)

	utxoTx := index.indexRemove[acc.GetPubKeyHash().String()]
	assert.Equal(t, len(tx.Vout), utxoTx.Size())
	for i := 0; i < len(tx.Vout); i++ {
		utxo := utxoTx.GetUtxo(tx.ID, i)
		assert.Equal(t, tx.Vout[i].Value, utxo.Value)
		assert.Equal(t, tx.Vout[i].Contract, utxo.Contract)
	}
}

func TestUTXOIndex_UndoTxsInBlock(t *testing.T) {
	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	db := storage.NewRamStorage()
	defer db.Close()

	acc := account.NewAccount()

	tx0 := transaction.Transaction{
		ID: []byte("test0"),
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(30),
				PubKeyHash: acc.GetPubKeyHash(),
				Contract:   "contract0",
			},
		},
	}
	err := transaction.PutTxJournal(tx0, db)
	assert.Nil(t, err)

	// 0. vin - should unspend tx0 from db
	// 1. vout - should be removed from index
	// 2. coinbase vin - should be ignored (remains in index)
	txs := []*transaction.Transaction{
		{
			ID: []byte("test1"),
			Vin: []transactionbase.TXInput{
				{
					Txid:      []byte("test0"),
					Vout:      0,
					Signature: nil,
					PubKey:    nil,
				},
			},
			Type: transaction.TxTypeNormal,
		},
		{
			ID: []byte("test2"),
			Vout: []transactionbase.TXOutput{
				{
					Value:      common.NewAmount(20),
					PubKeyHash: acc.GetPubKeyHash(),
					Contract:   "contract1",
				},
			},
			Type: transaction.TxTypeContract,
		},
		{
			ID: []byte("test3"),
			Vin: []transactionbase.TXInput{
				{
					Txid:      []byte("no"),
					Vout:      0,
					Signature: nil,
					PubKey:    nil,
				},
			},
			Type: transaction.TxTypeCoinbase,
		},
	}
	index.AddUTXO(txs[1].Vout[0], txs[1].ID, 0)
	err = index.Save()
	assert.Nil(t, err)

	blk := block.NewBlock(txs, nil, "producer")

	assert.Equal(t, 0, len(index.indexAdd))
	assert.Equal(t, 0, len(index.indexRemove))

	err = index.UndoTxsInBlock(blk, db)
	assert.Nil(t, err)

	utxoTxAdd := index.indexAdd[acc.GetPubKeyHash().String()]
	utxoTxRemove := index.indexRemove[acc.GetPubKeyHash().String()]

	assert.Equal(t, 1, utxoTxAdd.Size())
	assert.Equal(t, 1, utxoTxRemove.Size())

	utxosAdd := utxoTxAdd.GetAllUtxos()
	assert.Equal(t, 1, len(utxosAdd))

	assert.Equal(t, tx0.ID, utxosAdd[0].Txid)
	assert.Equal(t, tx0.Vout[0].Value, utxosAdd[0].Value)
	assert.Equal(t, tx0.Vout[0].PubKeyHash, utxosAdd[0].PubKeyHash)
	assert.Equal(t, tx0.Vout[0].Contract, utxosAdd[0].Contract)

	utxosRemove := utxoTxRemove.GetAllUtxos()
	assert.Equal(t, 1, len(utxosRemove))

	assert.Equal(t, txs[1].ID, utxosRemove[0].Txid)
	assert.Equal(t, txs[1].Vout[0].Value, utxosRemove[0].Value)
	assert.Equal(t, txs[1].Vout[0].PubKeyHash, utxosRemove[0].PubKeyHash)
	assert.Equal(t, txs[1].Vout[0].Contract, utxosRemove[0].Contract)
}
