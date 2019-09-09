package main

import (
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/util"
)

func main() {
	tx1 := transaction.Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  transactionbase.GenerateFakeTxInputs(),
		Vout: transactionbase.GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2),
	}
	txpb := tx1.ToProto()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			for i := 0; i < 1000000; i++ {
				tx := &transaction.Transaction{}
				tx.FromProto(txpb)
			}
		}
	}

}
