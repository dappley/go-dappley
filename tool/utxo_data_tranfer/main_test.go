package main

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/core"
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
	db := getDb()
	defer db.Close()

	minerKey := "dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"
	minerPubKey, ok := core.NewAddress(minerKey).GetPubKeyHash()
	if !ok {
		t.Error("ResultTest: wallet address is error!")
		return
	}
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
