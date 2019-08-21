package vm

import "C"
import (
	"strconv"
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

//export RecordRewardFunc
func RecordRewardFunc(handler unsafe.Pointer, address *C.char, amount *C.char) int {
	h := uint64(uintptr(handler))
	engine := getV8EngineByAddress(h)
	addr := C.GoString(address)
	amt := C.GoString(amount)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"reward_address": addr,
			"amount":         amt,
		}).Debug("SmartContract: failed to get smart engine handler!")
		return 1
	}

	if engine.rewards == nil {
		logger.WithFields(logger.Fields{
			"reward_address": addr,
			"amount":         amt,
		}).Debug("SmartContract: reward list has nil pointer!")
		return 1
	}

	if engine.rewards[addr] == "" {
		engine.rewards[addr] = "0"
	}

	originalAmt, err := strconv.Atoi(engine.rewards[addr])
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"reward_address":  addr,
			"original_amount": engine.rewards[addr],
		}).Warn("SmartContract: failed to access current reward list!")
		return 1
	}

	rewardAmt, err := strconv.Atoi(amt)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"reward_address": addr,
			"reward_amount":  amt,
		}).Warn("SmartContract: failed to read reward amount!")
		return 1
	}

	if originalAmt < 0 {
		logger.WithError(err).WithFields(logger.Fields{
			"reward_address":  addr,
			"original_amount": originalAmt,
		}).Warn("SmartContract: original amount is negative!")
		return 1
	}

	if rewardAmt <= 0 {
		logger.WithError(err).WithFields(logger.Fields{
			"reward_address": addr,
			"reward_amount":  rewardAmt,
		}).Warn("SmartContract: reward amount is negative!")
		return 1
	}

	engine.rewards[addr] = strconv.Itoa(originalAmt + rewardAmt)

	return 0
}
