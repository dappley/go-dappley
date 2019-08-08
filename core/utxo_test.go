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

package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/storage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddUTXO(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	txout := TXOutput{common.NewAmount(5), address1Hash, ""}
	utxoIndex := NewUTXOIndex(NewUTXOCache(storage.NewRamStorage()))

	utxoIndex.AddUTXO(txout, []byte{1}, 0)

	addr1UTXOs := utxoIndex.index[address1Hash.String()]
	assert.Equal(t, 1, addr1UTXOs.Size())
	assert.Equal(t, txout.Value, addr1UTXOs.GetAllUtxos()[0].Value)
	assert.Equal(t, []byte{1}, addr1UTXOs.GetAllUtxos()[0].Txid)
	assert.Equal(t, 0, addr1UTXOs.GetAllUtxos()[0].TxIndex)

	_, ok := utxoIndex.index["address2"]
	assert.Equal(t, false, ok)
}

func TestRemoveUTXO(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	utxoIndex := NewUTXOIndex(NewUTXOCache(storage.NewRamStorage()))

	addr1UtxoTx := NewUTXOTx()
	addr1UtxoTx.PutUtxo(&UTXO{TXOutput{common.NewAmount(5), address1Hash, ""}, []byte{1}, 0, UtxoNormal})
	addr1UtxoTx.PutUtxo(&UTXO{TXOutput{common.NewAmount(2), address1Hash, ""}, []byte{1}, 1, UtxoNormal})
	addr1UtxoTx.PutUtxo(&UTXO{TXOutput{common.NewAmount(2), address1Hash, ""}, []byte{2}, 0, UtxoNormal})

	addr2UtxoTx := NewUTXOTx()
	addr2UtxoTx.PutUtxo(&UTXO{TXOutput{common.NewAmount(4), address2Hash, ""}, []byte{1}, 2, UtxoNormal})

	utxoIndex.index[address1Hash.String()] = &addr1UtxoTx
	utxoIndex.index[address2Hash.String()] = &addr2UtxoTx

	err := utxoIndex.removeUTXO(address1Hash, []byte{1}, 0)

	assert.Nil(t, err)
	assert.Equal(t, 2, utxoIndex.index[address1Hash.String()].Size())
	assert.Equal(t, 1, utxoIndex.index[address2Hash.String()].Size())

	err = utxoIndex.removeUTXO(address2Hash, []byte{2}, 1) // Does not exists

	assert.NotNil(t, err)
	assert.Equal(t, 2, utxoIndex.index[address1Hash.String()].Size())
	assert.Equal(t, 1, utxoIndex.index[address2Hash.String()].Size())
}

func TestUpdate(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	blk := GenerateUtxoMockBlockWithoutInputs()
	utxoIndex := NewUTXOIndex(NewUTXOCache(db))
	utxoIndex.UpdateUtxoState(blk.GetTransactions())
	utxoIndex.Save()
	utxoIndexInDB := NewUTXOIndex(NewUTXOCache(db))

	// test updating UTXO index with non-dependent transactions
	// Assert that both the original instance and the database copy are updated correctly
	for _, index := range []UTXOIndex{*utxoIndex, *utxoIndexInDB} {
		utxoTx := index.GetAllUTXOsByPubKeyHash(address1Hash)
		assert.Equal(t, 2, utxoTx.Size())
		utxo0 := utxoTx.GetUtxo(blk.GetTransactions()[0].ID, 0)
		utx1 := utxoTx.GetUtxo(blk.GetTransactions()[0].ID, 1)
		assert.Equal(t, blk.GetTransactions()[0].ID, utxo0.Txid)
		assert.Equal(t, 0, utxo0.TxIndex)
		assert.Equal(t, blk.GetTransactions()[0].Vout[0].Value, utxo0.Value)
		assert.Equal(t, blk.GetTransactions()[0].ID, utx1.Txid)
		assert.Equal(t, 1, utx1.TxIndex)
		assert.Equal(t, blk.GetTransactions()[0].Vout[1].Value, utx1.Value)
	}

	// test updating UTXO index with dependent transactions
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = account.GenerateKeyPairByPrivateKey(prikey1).GetPublicKey()
	var pkHash1, _ = account.NewUserPubKeyHash(pubkey1)
	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var pubkey2 = account.GenerateKeyPairByPrivateKey(prikey2).GetPublicKey()
	var pkHash2, _ = account.NewUserPubKeyHash(pubkey2)
	var prikey3 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa73"
	var pubkey3 = account.GenerateKeyPairByPrivateKey(prikey3).GetPublicKey()
	var pkHash3, _ = account.NewUserPubKeyHash(pubkey3)
	var prikey4 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa74"
	var pubkey4 = account.GenerateKeyPairByPrivateKey(prikey4).GetPublicKey()
	var pkHash4, _ = account.NewUserPubKeyHash(pubkey4)
	var prikey5 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa75"
	var pubkey5 = account.GenerateKeyPairByPrivateKey(prikey5).GetPublicKey()
	var pkHash5, _ = account.NewUserPubKeyHash(pubkey5)

	var dependentTx1 = Transaction{
		ID: nil,
		Vin: []TXInput{
			{tx1.ID, 1, nil, pubkey1},
		},
		Vout: []TXOutput{
			{common.NewAmount(5), pkHash1, ""},
			{common.NewAmount(10), pkHash2, ""},
		},
		Tip: common.NewAmount(3),
	}
	dependentTx1.ID = dependentTx1.Hash()

	var dependentTx2 = Transaction{
		ID: nil,
		Vin: []TXInput{
			{dependentTx1.ID, 1, nil, pubkey2},
		},
		Vout: []TXOutput{
			{common.NewAmount(5), pkHash3, ""},
			{common.NewAmount(3), pkHash4, ""},
		},
		Tip: common.NewAmount(2),
	}
	dependentTx2.ID = dependentTx2.Hash()

	var dependentTx3 = Transaction{
		ID: nil,
		Vin: []TXInput{
			{dependentTx2.ID, 0, nil, pubkey3},
		},
		Vout: []TXOutput{
			{common.NewAmount(1), pkHash4, ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx3.ID = dependentTx3.Hash()

	var dependentTx4 = Transaction{
		ID: nil,
		Vin: []TXInput{
			{dependentTx2.ID, 1, nil, pubkey4},
			{dependentTx3.ID, 0, nil, pubkey4},
		},
		Vout: []TXOutput{
			{common.NewAmount(3), pkHash1, ""},
		},
		Tip: common.NewAmount(1),
	}
	dependentTx4.ID = dependentTx4.Hash()

	var dependentTx5 = Transaction{
		ID: nil,
		Vin: []TXInput{
			{dependentTx1.ID, 0, nil, pubkey1},
			{dependentTx4.ID, 0, nil, pubkey1},
		},
		Vout: []TXOutput{
			{common.NewAmount(4), pkHash5, ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx5.ID = dependentTx5.Hash()

	utxoPk2 := &UTXO{dependentTx1.Vout[1], dependentTx1.ID, 1, UtxoNormal}
	utxoPk1 := &UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, UtxoNormal}

	utxoTxPk2 := NewUTXOTx()
	utxoTxPk2.PutUtxo(utxoPk2)

	utxoTxPk1 := NewUTXOTx()
	utxoTxPk1.PutUtxo(utxoPk1)

	utxoIndex2 := NewUTXOIndex(NewUTXOCache(storage.NewRamStorage()))

	utxoIndex2.index[pkHash2.String()] = &utxoTxPk2
	utxoIndex2.index[pkHash1.String()] = &utxoTxPk1

	tx2Utxo1 := UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, UtxoNormal}
	tx2Utxo2 := UTXO{dependentTx2.Vout[1], dependentTx2.ID, 1, UtxoNormal}
	tx2Utxo3 := UTXO{dependentTx3.Vout[0], dependentTx3.ID, 0, UtxoNormal}
	tx2Utxo4 := UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, UtxoNormal}
	tx2Utxo5 := UTXO{dependentTx4.Vout[0], dependentTx4.ID, 0, UtxoNormal}
	dependentTx2.Sign(account.GenerateKeyPairByPrivateKey(prikey2).GetPrivateKey(), utxoIndex2.index[pkHash2.String()].GetAllUtxos())
	dependentTx3.Sign(account.GenerateKeyPairByPrivateKey(prikey3).GetPrivateKey(), []*UTXO{&tx2Utxo1})
	dependentTx4.Sign(account.GenerateKeyPairByPrivateKey(prikey4).GetPrivateKey(), []*UTXO{&tx2Utxo2, &tx2Utxo3})
	dependentTx5.Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), []*UTXO{&tx2Utxo4, &tx2Utxo5})

	txsForUpdate := []*Transaction{&dependentTx2, &dependentTx3}
	utxoIndex2.UpdateUtxoState(txsForUpdate)
	assert.Equal(t, 1, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash1).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash2).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash3).Size())
	assert.Equal(t, 2, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash4).Size())
	txsForUpdate = []*Transaction{&dependentTx2, &dependentTx3, &dependentTx4}
	utxoIndex2.UpdateUtxoState(txsForUpdate)
	assert.Equal(t, 2, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash1).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash2).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash3).Size())
	txsForUpdate = []*Transaction{&dependentTx2, &dependentTx3, &dependentTx4, &dependentTx5}
	utxoIndex2.UpdateUtxoState(txsForUpdate)
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash1).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash2).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash3).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash4).Size())
	assert.Equal(t, 1, utxoIndex2.GetAllUTXOsByPubKeyHash(pkHash5).Size())
}

func TestUpdate_Failed(t *testing.T) {
	db := new(mocks.Storage)

	simulatedFailure := errors.New("simulated storage failure")
	db.On("Put", mock.Anything, mock.Anything).Return(simulatedFailure)
	db.On("Get", mock.Anything, mock.Anything).Return(nil, nil)

	blk := GenerateUtxoMockBlockWithoutInputs()
	utxoIndex := NewUTXOIndex(NewUTXOCache(db))
	utxoIndex.UpdateUtxoState(blk.GetTransactions())
	err := utxoIndex.Save()
	assert.Equal(t, simulatedFailure, err)
	assert.Equal(t, 2, utxoIndex.GetAllUTXOsByPubKeyHash(address1Hash).Size())
}

func TestFindUTXO(t *testing.T) {
	Txin := MockTxInputs()
	Txin = append(Txin, MockTxInputs()...)
	utxo1 := &UTXO{TXOutput{common.NewAmount(10), account.PubKeyHash([]byte("addr1")), ""}, Txin[0].Txid, Txin[0].Vout, UtxoNormal}
	utxo2 := &UTXO{TXOutput{common.NewAmount(9), account.PubKeyHash([]byte("addr1")), ""}, Txin[1].Txid, Txin[1].Vout, UtxoNormal}
	utxoTx1 := NewUTXOTxWithData(utxo1)
	utxoTx2 := NewUTXOTxWithData(utxo2)

	assert.Equal(t, utxo1, utxoTx1.GetUtxo(Txin[0].Txid, Txin[0].Vout))
	assert.Equal(t, utxo2, utxoTx2.GetUtxo(Txin[1].Txid, Txin[1].Vout))
	assert.Nil(t, utxoTx1.GetUtxo(Txin[2].Txid, Txin[2].Vout))
	assert.Nil(t, utxoTx2.GetUtxo(Txin[3].Txid, Txin[3].Vout))
}

func TestConcurrentUTXOindexReadWrite(t *testing.T) {
	index := NewUTXOIndex(NewUTXOCache(storage.NewRamStorage()))

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
					index.AddUTXO(TXOutput{}, []byte("asd"), 65)
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

	contractPkh := account.NewContractPubKeyHash()
	//preapre 3 utxos in the utxo index
	txoutputs := []TXOutput{
		{common.NewAmount(3), address1Hash, ""},
		{common.NewAmount(4), address2Hash, ""},
		{common.NewAmount(5), address2Hash, ""},
		{common.NewAmount(2), contractPkh, "helloworld!"},
		{common.NewAmount(4), contractPkh, ""},
	}

	index := NewUTXOIndex(NewUTXOCache(storage.NewRamStorage()))
	for i, txoutput := range txoutputs {
		index.AddUTXO(txoutput, []byte("01"), i)
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
			[]byte(address2Hash),
			nil},

		{"notEnoughUtxo",
			common.NewAmount(4),
			[]byte(address1Hash),
			ErrInsufficientFund},

		{"justEnoughUtxo",
			common.NewAmount(9),
			[]byte(address2Hash),
			nil},
		{"notEnoughUtxo2",
			common.NewAmount(10),
			[]byte(address2Hash),
			ErrInsufficientFund},
		{"smartContractUtxo",
			common.NewAmount(3),
			[]byte(contractPkh),
			nil},
		{"smartContractUtxoInsufficient",
			common.NewAmount(5),
			[]byte(contractPkh),
			ErrInsufficientFund},
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
	utxoIndex := NewUTXOIndex(NewUTXOCache(storage.NewRamStorage()))
	utxoCopy := utxoIndex.DeepCopy()
	assert.Equal(t, 0, len(utxoIndex.index))
	assert.Equal(t, 0, len(utxoCopy.index))

	addr1UtxoTx := NewUTXOTx()
	utxoIndex.index[string(address1Hash)] = &addr1UtxoTx
	assert.Equal(t, 1, len(utxoIndex.index))
	assert.Equal(t, 0, len(utxoCopy.index))

	copyUtxoTx := NewUTXOTxWithData(&UTXO{MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, UtxoNormal})
	utxoCopy.index[string(address1Hash)] = &copyUtxoTx
	assert.Equal(t, 1, len(utxoIndex.index))
	assert.Equal(t, 1, len(utxoCopy.index))
	assert.Equal(t, 0, utxoIndex.index[string(address1Hash)].Size())
	assert.Equal(t, 1, utxoCopy.index[string(address1Hash)].Size())

	copyUtxoTx1 := NewUTXOTx()
	copyUtxoTx1.PutUtxo(&UTXO{MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, UtxoNormal})
	copyUtxoTx1.PutUtxo(&UTXO{MockUtxoOutputsWithoutInputs()[1], []byte{}, 1, UtxoNormal})
	utxoCopy.index["1"] = &copyUtxoTx1

	utxoCopy2 := utxoCopy.DeepCopy()
	copy2UtxoTx1 := NewUTXOTx()
	copy2UtxoTx1.PutUtxo(&UTXO{MockUtxoOutputsWithoutInputs()[0], []byte{}, 0, UtxoNormal})
	utxoCopy2.index["1"] = &copy2UtxoTx1
	assert.Equal(t, 2, len(utxoCopy.index))
	assert.Equal(t, 2, len(utxoCopy2.index))
	assert.Equal(t, 2, utxoCopy.index["1"].Size())
	assert.Equal(t, 1, utxoCopy2.index["1"].Size())
	assert.Equal(t, 1, len(utxoIndex.index))

	assert.EqualValues(t, utxoCopy.index[address1Hash.String()], utxoCopy2.index[address1Hash.String()])
}
