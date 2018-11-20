package sc

import "C"
import (
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

type LogFunc func(...interface{})

const (
	DEBUG uint32 = 0
	INFO  uint32 = 1
	WARN  uint32 = 2
	ERROR uint32 = 3
)

var logFuncEntries = map[uint32]LogFunc{
	DEBUG: logger.Debug,
	INFO:  logger.Info,
	WARN:  logger.Warn,
	ERROR: logger.Error,
}

//export LoggerFunc
func LoggerFunc(level C.uint, args **C.char, length C.int) {
	logFunc, ok := logFuncEntries[uint32(level)]
	if ok == false {
		logger.WithFields(logger.Fields{
			"level": uint32(level),
		}).Info("Smart Contract")
		return
	}

	argSlice := (*[1 << 30]*C.char)(unsafe.Pointer(args))[:length:length]
	goArgs := make([]string, length+1)
	goArgs = append(goArgs, "[Contract]")
	for _, arg := range argSlice {
		goArgs = append(goArgs, C.GoString(arg))
	}

	logFunc(goArgs)
}
