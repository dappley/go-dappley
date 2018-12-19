package vm

import "C"
import (
	"math/rand"
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

//export RandomFunc
func RandomFunc(handler unsafe.Pointer, max C.int) int {
	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.WithFields(logger.Fields{
			"handler":  uint64(uintptr(handler)),
			"function": "Math.RandomFunc",
			"max":      max,
		}).Debug("SmartContract: failed to get the engine instance!")
		return -1
	}

	if engine.seed == 0 {
		logger.WithFields(logger.Fields{
			"handler":  uint64(uintptr(handler)),
			"function": "Math.RandomFunc",
			"max":      max,
		}).Debug("SmartContract: failed to get random seed!")
		return -1
	}

	rand.Seed(engine.seed)
	return rand.Intn(int(max))
}
