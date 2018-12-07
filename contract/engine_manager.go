package vm

import (
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

const scheduleFuncName = "dapp_schedule"

type V8EngineManager struct {
	address core.Address
}

func NewV8EngineManager(address core.Address) *V8EngineManager {
	return &V8EngineManager{address}
}

func (em *V8EngineManager) CreateEngine() core.ScEngine {
	engine := NewV8Engine()
	engine.ImportNodeAddress(em.address)
	return engine
}

func (em *V8EngineManager) RunScheduledEvents(contractUtxos []*core.UTXO,
	scStorage *core.ScState,
	blkHeight uint64,
	seed int64) {
	logger.WithFields(logger.Fields{
		"numOfSmartContract": len(contractUtxos),
	}).Info("Running Scheduled Events...")

	for _, utxo := range contractUtxos {
		addr := utxo.PubKeyHash.GenerateAddress()
		engine := em.CreateEngine()
		engine.ImportSourceCode(utxo.Contract)
		engine.ImportLocalStorage(scStorage.GetStorageByAddress(addr.String()))
		engine.ImportContractAddr(addr)
		engine.ImportSourceTXID(utxo.Txid)
		engine.ImportCurrBlockHeight(blkHeight)
		engine.ImportSeed(seed)
		engine.Execute(scheduleFuncName, "")
		engine.DestroyEngine()
	}
}
