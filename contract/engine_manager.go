package vm

import (
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

const scheduleFuncName = "dapp_schedule"

type V8EngineManager struct{}

func NewV8EngineManager() *V8EngineManager{
	return &V8EngineManager{}
}

func (em *V8EngineManager) CreateEngine() core.ScEngine{
	return NewV8Engine()
}

func (em *V8EngineManager) RunScheduledEvents(contractUtxos []*core.UTXO, scStorage *core.ScState, ){
	logger.WithFields(logger.Fields{
		"numOfSmartContract" : len(contractUtxos),
	}).Info("Running Scheduled Events...")

	for _, utxo := range contractUtxos{
		addr := utxo.PubKeyHash.GenerateAddress()
		engine := NewV8Engine()
		engine.ImportSourceCode(utxo.Contract)
		engine.ImportLocalStorage(scStorage.GetStorageByAddress(addr.String()))
		engine.ImportContractAddr(addr)
		engine.ImportSourceTXID(utxo.Txid)
		engine.Execute(scheduleFuncName,"")
	}
}