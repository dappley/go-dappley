package vm

import (
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic/ltransaction"
)

const scheduleFuncName = "dapp_schedule"

type V8EngineManager struct {
	address account.Address
}

func NewV8EngineManager(address account.Address) *V8EngineManager {
	return &V8EngineManager{address}
}

func (em *V8EngineManager) CreateEngine() ltransaction.ScEngine {
	engine := NewV8Engine()
	engine.ImportNodeAddress(em.address)
	return engine
}
