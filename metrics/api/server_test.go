package metrics

import (
    "testing"

    "github.com/dappley/go-dappley/core"
    "github.com/dappley/go-dappley/storage"
    logger "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
)

func init() {
    InitAPI()
    logger.SetLevel(logger.PanicLevel)
}

func TestStartAPI(t *testing.T) {
    size, err := GetTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t,0, size)
}

func TestTransactionPoolSize(t *testing.T) {
    // add transaction
    txPool := core.NewTransactionPool(1)
    tx := core.MockTransaction()
    txPool.Push(*tx)
    size, err := GetTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 1, size)

    // exceed tx pool limit
    txPool.Push(*core.MockTransaction())
    size, err = GetTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 1, size)

    // verify deserialization restores metric
    ramStorage := storage.NewRamStorage()
    err = txPool.SaveToDatabase(ramStorage)
    assert.Nil(t, err)
    newTXPool := core.LoadTxPoolFromDatabase(ramStorage, 1)
    size, err = GetTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 1, newTXPool.GetNumOfTxInPool())
    assert.Equal(t, 1, size)

    // remove transaction from pool
    txPool.CleanUpMinedTxs([]*core.Transaction{tx})
    size, err = GetTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 0, size)
}
