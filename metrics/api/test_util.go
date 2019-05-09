package metrics

import (
    "encoding/json"
    "fmt"
    "net/http"
)

const (
    maxRetries = 3
)

var (
    metricsURL string
)

type response struct {
    TransactionPoolSize int `json:"dap.txPool.currSize"`
}

func InitAPI() {
    if metricsURL == "" {
        metricsURL = fmt.Sprintf("http://localhost:%d/debug/metrics", StartAPI(0))
    }
}

func GetTransactionPoolSize() (int, error) {

    var resp *http.Response
    var err error
    for i := 0; i < maxRetries; i++ {
        resp, err = http.Get(metricsURL)
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