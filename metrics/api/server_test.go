package metrics

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "testing"

    "github.com/dappley/go-dappley/core"
    "github.com/dappley/go-dappley/storage"
    logger "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
)

const (
    MaxRetries = 3
)

var (
    URL string
)

type response struct {
    TransactionPoolSize int `json:"dap.txPool.currSize"`
}

func TestMain(m *testing.M) {
    initAPI()
    logger.SetLevel(logger.PanicLevel)
    retCode := m.Run()
    os.Exit(retCode)
}

func TestStartAPI(t *testing.T) {
    size, err := getTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t,0, size)
}

func TestTransactionPoolSize(t *testing.T) {
    // add transaction
    txPool := core.NewTransactionPool(1)
    tx := core.MockTransaction()
    txPool.Push(*tx)
    size, err := getTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 1, size)

    // exceed tx pool limit
    txPool.Push(*core.MockTransaction())
    size, err = getTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 1, size)

    // verify deserialization restores metric
    ramStorage := storage.NewRamStorage()
    err = txPool.SaveToDatabase(ramStorage)
    assert.Nil(t, err)
    newTXPool := core.LoadTxPoolFromDatabase(ramStorage, 1)
    size, err = getTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 1, newTXPool.GetNumOfTxInPool())
    assert.Equal(t, 1, size)

    // remove transaction from pool
    txPool.CleanUpMinedTxs([]*core.Transaction{tx})
    size, err = getTransactionPoolSize()
    assert.Nil(t, err)
    assert.Equal(t, 0, size)
}

func initAPI() {
    if URL == "" {
        URL = fmt.Sprintf("http://localhost:%d/debug/metrics", StartAPI(0))
    }
}

func getTransactionPoolSize() (int, error) {

    var resp *http.Response
    var err error
    for i := 0; i < MaxRetries; i++ {

        resp, err = http.Get(URL)
        if err == nil {
            break
        }
    }

    if err != nil {
        return -1, err
    }

    response := &response{}
    err = json.NewDecoder(resp.Body).Decode(response)
    if err != nil {
        return -1, err
    }

    return response.TransactionPoolSize, nil
}