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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Padding Address to 32 Byte
var address1Bytes = []byte("address1000000000000000000000000")
var address2Bytes = []byte("address2000000000000000000000000")
var ta1 = account.NewTransactionAccountByPubKey(address1Bytes)
var ta2 = account.NewTransactionAccountByPubKey(address2Bytes)

func TestAddUTXO(t *testing.T) {
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

func TestRemoveUTXO(t *testing.T) {
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

func TestUTXOIndex_GetUTXOsByAmount(t *testing.T) {
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

	index := NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
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
			utxos, err := index.GetUTXOsByAmount(tt.pubKey, tt.amount)
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

func TestUTXOIndexSave(t *testing.T) {

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
	utxoIndex2.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx1, //first time add utxoPk10
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
