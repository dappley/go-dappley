package vm

import "C"
import (
	logger "github.com/sirupsen/logrus"
	"math/rand"
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

	if engine.seed == 0{
		logger.WithFields(logger.Fields{
			"handler" : uint64(uintptr(handler)),
			"function": "Math.random",
			"max":  max,
		}).Debug("Smart Contract: Failed to get random seed!")
		return -1
	}

	rand.Seed(engine.seed)
	return rand.Intn(int(max))
}