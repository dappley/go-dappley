package vm

import "github.com/dappley/go-dappley/core"

type V8EngineManager struct{}

func NewV8EngineManager() *V8EngineManager{
	return &V8EngineManager{}
}

func (em *V8EngineManager) CreateEngine() core.ScEngine{
	return NewV8Engine()
}