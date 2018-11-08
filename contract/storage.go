package sc
import "C"
import (
	logger "github.com/sirupsen/logrus"
	"unsafe"
)


//export StorageGetFunc
func StorageGetFunc(address unsafe.Pointer, key *C.char) *C.char{
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)
	goKey := C.GoString(key)

	if engine == nil{
		logger.WithFields(logger.Fields{
			"contractAddr"		: addr,
			"key"	  			: goKey,
		}).Debug("Smart Contract: Failed to get storage handler!")
		return nil
	}

	val := engine.storage[goKey]
	if val == "" {
		logger.WithFields(logger.Fields{
			"contractAddr"		: addr,
			"key"	  			: goKey,
		}).Debug("Smart Contract: Failed to get value from storage")
		return nil
	}

	return C.CString(val)
}

//export StorageSetFunc
func StorageSetFunc(address unsafe.Pointer, key,value *C.char) int{
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)
	goKey := C.GoString(key)
	goVal := C.GoString(value)

	if engine == nil{
		logger.WithFields(logger.Fields{
			"contractAddr"		: addr,
			"key"	  			: goKey,
		}).Debug("Smart Contract: Failed to get storage handler!")
		return 1
	}

	engine.storage[goKey] = goVal
	return 0
}

//export StorageDelFunc
func StorageDelFunc(address unsafe.Pointer, key *C.char) int{
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)
	goKey := C.GoString(key)

	if engine == nil{
		logger.WithFields(logger.Fields{
			"contractAddr"		: addr,
			"key"	  			: goKey,
		}).Debug("Smart Contract: Failed to get storage handler!")
		return 1
	}
	delete(engine.storage, goKey)
	return 0
}