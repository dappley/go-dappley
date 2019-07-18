package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/util"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func GenerateFakeTxInputs() []core.TXInput {
	return []core.TXInput{
		{getAoB(2), 10, getAoB(2), getAoB(2)},
		{getAoB(2), 5, getAoB(2), getAoB(2)},
	}
}

func GenerateFakeTxOutputs() []core.TXOutput {
	return []core.TXOutput{
		{common.NewAmount(1), client.PubKeyHash(getAoB(2)), ""},
		{common.NewAmount(2), client.PubKeyHash(getAoB(2)), ""},
	}
}

func main() {
	tx1 := core.Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
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
				tx := &core.Transaction{}
				tx.FromProto(txpb)
			}
		}
	}

}
