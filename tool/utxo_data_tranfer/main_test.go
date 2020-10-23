package main

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/core/transaction"
	"os"
	"testing"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

// Test whether data format transfer is succeed.
func TestTransferResult(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	minerKey := "dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"
	minerAccount := account.NewTransactionAccountByAddress(account.NewAddress(minerKey))

	if !minerAccount.IsValid() {
		t.Error("ResultTest: account address is error!")
		return
	}
	minerPubKey := minerAccount.GetPubKeyHash()
	// put old data
	txid1, _ := hex.DecodeString("948c984f0cdcefc4f977efcd93ae37360cc5165dfc3657f07e72306cd0e6a354")
	txid2, _ := hex.DecodeString("4fef1c385b0cbda4092cfe245329bb18e580480e07a880ebcefe1fa7e24a089f")
	utxo1 := &utxo.UTXO{transactionbase.TXOutput{transaction.Subsidy, minerPubKey, ""}, txid1, 0, utxo.UtxoNormal,[]byte{}}
	utxo2 := &utxo.UTXO{transactionbase.TXOutput{transaction.Subsidy, minerPubKey, ""}, txid2, 0, utxo.UtxoNormal,[]byte{}}
	utxos := []*utxo.UTXO{utxo1, utxo2}
	utxoIndexOld := NewUTXOIndexOld()
	utxoIndexOld.index[minerKey] = utxos

	t.Logf("ResultTest: utxoIndexOld Size %d", len(utxoIndexOld.index[minerKey]))
	utxoBytes := utxoIndexOld.serializeUTXOIndexOld()
	db.Put([]byte(utxoMapKeyOld), utxoBytes)

	// convert to new data
	convert(db)

	// read new data
	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(db))
	utxoTx := utxoIndex.GetAllUTXOsByPubKeyHash(minerPubKey)
	newDataSize := utxoTx.Size()
	t.Logf("ResultTest: newDataSize %d", newDataSize)
	//_, utxo, nextUtxoTx := utxoTx.Iterator()
	for _, utxo := range utxoTx.Indices {
		t.Logf("ResultTest: txid:%v, txIndex:%d", hex.EncodeToString(utxo.Txid), utxo.TxIndex)
		//_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}
	assert.True(t, newDataSize > 0, "Data convert and save failed")
}
