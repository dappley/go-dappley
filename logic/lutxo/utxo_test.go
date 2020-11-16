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
	addr1UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(5), ta1.GetPubKeyHash(), ""}, []byte{1}, 0, utxo.UtxoNormal, []byte{}})
	addr1UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(2), ta1.GetPubKeyHash(), ""}, []byte{1}, 1, utxo.UtxoNormal, []byte{}})
	addr1UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(2), ta1.GetPubKeyHash(), ""}, []byte{2}, 0, utxo.UtxoNormal, []byte{}})

	addr2UtxoTx := utxo.NewUTXOTx()
	addr2UtxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(4), ta2.GetPubKeyHash(), ""}, []byte{1}, 2, utxo.UtxoNormal, []byte{}})

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
	utxo1 := &utxo.UTXO{transactionbase.TXOutput{common.NewAmount(10), account.PubKeyHash([]byte("addr1")), ""}, Txin[0].Txid, Txin[0].Vout, utxo.UtxoNormal, []byte{}}
	utxo2 := &utxo.UTXO{transactionbase.TXOutput{common.NewAmount(9), account.PubKeyHash([]byte("addr1")), ""}, Txin[1].Txid, Txin[1].Vout, utxo.UtxoNormal, []byte{}}
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

	copyUtxoTx := utxo.NewUTXOTxWithData(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, utxo.UtxoNormal, []byte{}})
	utxoCopy.indexAdd[string(ta1.GetPubKeyHash())] = &copyUtxoTx
	assert.Equal(t, 1, len(utxoIndex.indexAdd))
	assert.Equal(t, 1, len(utxoCopy.indexAdd))
	assert.Equal(t, 0, utxoIndex.indexAdd[string(ta1.GetPubKeyHash())].Size())
	assert.Equal(t, 1, utxoCopy.indexAdd[string(ta1.GetPubKeyHash())].Size())

	copyUtxoTx1 := utxo.NewUTXOTx()
	copyUtxoTx1.PutUtxo(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, utxo.UtxoNormal, []byte{}})
	copyUtxoTx1.PutUtxo(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[1], []byte{}, 1, utxo.UtxoNormal, []byte{}})
	utxoCopy.indexAdd["1"] = &copyUtxoTx1

	utxoCopy2 := utxoCopy.DeepCopy()
	copy2UtxoTx1 := utxo.NewUTXOTx()
	copy2UtxoTx1.PutUtxo(&utxo.UTXO{core.MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, utxo.UtxoNormal, []byte{}})
	utxoCopy2.indexAdd["1"] = &copy2UtxoTx1
	assert.Equal(t, 2, len(utxoCopy.indexAdd))
	assert.Equal(t, 2, len(utxoCopy2.indexAdd))
	assert.Equal(t, 2, utxoCopy.indexAdd["1"].Size())
	assert.Equal(t, 1, utxoCopy2.indexAdd["1"].Size())
	assert.Equal(t, 1, len(utxoIndex.indexAdd))

	assert.EqualValues(t, utxoCopy.indexAdd[ta1.GetPubKeyHash().String()], utxoCopy2.indexAdd[ta1.GetPubKeyHash().String()])
}
