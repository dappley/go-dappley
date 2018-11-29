package vm

import "C"
import (
	logger "github.com/sirupsen/logrus"
	"math/rand"
	"time"
	"unsafe"
)

//export RandomFunc
func RandomFunc(handler unsafe.Pointer, max C.int) int{
	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.WithFields(logger.Fields{
			"handler" : uint64(uintptr(handler)),
			"function": "Math.random",
			"max":  max,
		}).Debug("Smart Contract: Failed to get the engine instance while executing transfer!")
		return -1
	}
	logger.WithFields(logger.Fields{
		"max": max,
	}).Info("Smart Contract: random function has been called")

	if engine.sourceTXID == nil{
		logger.WithFields(logger.Fields{
			"handler" : uint64(uintptr(handler)),
			"function": "Math.random",
			"max":  max,
		}).Debug("Smart Contract: Failed to get source txid!")
		return -1
	}

	rand.Seed(time.Now().UnixNano())
	return rand.Intn(int(max))
}