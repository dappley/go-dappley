package vm

import (
	logger "github.com/sirupsen/logrus"
	"strings"

	"github.com/dappley/go-dappley/core"
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
		"smart_contracts": len(contractUtxos),
	}).Info("V8EngineManager: is running scheduled events...")

	for _, utxo := range contractUtxos {
		if !strings.Contains(utxo.Contract, scheduleFuncName){
			continue
		}
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
