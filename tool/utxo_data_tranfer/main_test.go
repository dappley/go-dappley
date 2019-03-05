package main

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
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
	minerPubKey, ok := core.NewAddress(minerKey).GetPubKeyHash()
	if !ok {
		t.Error("ResultTest: wallet address is error!")
		return
	}
	// put old data
	Txin := core.MockTxInputs()
	Txin = append(Txin, core.MockTxInputs()...)
	utxo1 := &core.UTXO{core.TXOutput{common.NewAmount(10), minerPubKey, ""}, Txin[0].Txid, Txin[0].Vout, core.UtxoNormal}
	utxo2 := &core.UTXO{core.TXOutput{common.NewAmount(9), minerPubKey, ""}, Txin[1].Txid, Txin[1].Vout, core.UtxoNormal}
	utxos := []*core.UTXO{utxo1, utxo2}
	utxoIndexOld := NewUTXOIndexOld()
	utxoIndexOld.index[minerKey] = utxos

	t.Logf("ResultTest: utxoIndexOld Size %d", len(utxoIndexOld.index[minerKey]))
	utxoBytes := utxoIndexOld.serializeUTXOIndexOld()
	db.Put([]byte(utxoMapKeyOld), utxoBytes)

	// convert to new data
	convert(db)

	// read new data
	utxoIndex := core.NewUTXOIndex(core.NewUTXOCache(db))
	utxoTx := utxoIndex.GetAllUTXOsByPubKeyHash(minerPubKey)
	newDataSize := utxoTx.Size()
	t.Logf("ResultTest: newDataSize %d", newDataSize)
	_, utxo, nextUtxoTx := utxoTx.Iterator();
	for utxo != nil {
		t.Logf("ResultTest: txid:%v, txIndex:%d", hex.EncodeToString(utxo.Txid), utxo.TxIndex)
		_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}
	assert.True(t, newDataSize > 0, "Data convert and save failed")
}
