package sc
import "C"
import (
	logger "github.com/sirupsen/logrus"
	"strconv"
	"unsafe"
)

//export RecordRewardFunc
func RecordRewardFunc(handler unsafe.Pointer, address *C.char, amount *C.char) int{
	h := uint64(uintptr(handler))
	engine := getV8EngineByAddress(h)
	addr := C.GoString(address)
	amt := C.GoString(amount)

	if engine == nil{
		logger.WithFields(logger.Fields{
			"reward address"		: addr,
			"amount"	  			: amt,
		}).Debug("Smart Contract: Failed to get storage handler!")
		return 1
	}

	if engine.storage[addr] == "" {
		engine.storage[addr] = "0"
	}

	originalAmt, err := strconv.Atoi(engine.storage[addr])
	if err != nil{
		logger.WithFields(logger.Fields{
			"reward address"		: addr,
			"original amount"	  	: engine.storage[addr],
			"error"					: err,
		}).Warn("Smart Contract: Current reward list access failed!")
		return 1
	}

	rewardAmt, err := strconv.Atoi(amt)
	if err != nil{
		logger.WithFields(logger.Fields{
			"reward address"		: addr,
			"reward amount"	  		: amt,
			"error"					: err,
		}).Warn("Smart Contract: Read reward amount failed!")
		return 1
	}

	if originalAmt < 0 {
		logger.WithFields(logger.Fields{
			"reward address"		: addr,
			"original amount"		: originalAmt,
			"error"					: err,
		}).Warn("Smart Contract: Original Amount is negative!")
		return 1
	}

	if rewardAmt <= 0 {
		logger.WithFields(logger.Fields{
			"reward address"		: addr,
			"reward amount"			: rewardAmt,
			"error"					: err,
		}).Warn("Smart Contract: Reward Amount is negative!")
		return 1
	}


	engine.storage[addr] = strconv.Itoa(originalAmt+rewardAmt)

	return 0
}
