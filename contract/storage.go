package vm

import "C"
import (
	"errors"
	"regexp"
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

var (
	// StorageKeyPattern the pattern of varible key stored in stateDB
	/*
		const fieldNameRe = /^[a-zA-Z_$][a-zA-Z0-9_]+$/;
		var combineStorageMapKey = function (fieldName, key) {
			return "@" + fieldName + "[" + key + "]";
		};
	*/
	StorageKeyPattern = regexp.MustCompile("^@([a-zA-Z_$][a-zA-Z0-9_]+?)\\[(.*?)\\]$")
	// DefaultDomainKey the default domain key
	DefaultDomainKey = "_"
	// ErrInvalidStorageKey invalid storage key error
	ErrInvalidStorageKey = errors.New("invalid storage key")
)

//export StorageGetFunc
func StorageGetFunc(address unsafe.Pointer, key *C.char) *C.char {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)
	goKey := C.GoString(key)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
			"key":              goKey,
		}).Debug("SmartContract: failed to get storage handler!")
		return nil
	}

	val := engine.storage[goKey]
	if val == "" {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
			"key":              goKey,
		}).Debug("SmartContract: failed to get value from storage.")
		return nil
	}

	return C.CString(val)
}

//export StorageSetFunc
func StorageSetFunc(address unsafe.Pointer, key, value *C.char) int {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)
	goKey := C.GoString(key)
	goVal := C.GoString(value)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
			"key":              goKey,
		}).Debug("SmartContract: failed to get storage handler!")
		return 1
	}

	engine.storage[goKey] = goVal
	return 0
}

//export StorageDelFunc
func StorageDelFunc(address unsafe.Pointer, key *C.char) int {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)
	goKey := C.GoString(key)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
			"key":              goKey,
		}).Debug("SmartContract: failed to get storage handler!")
		return 1
	}
	delete(engine.storage, goKey)
	return 0
}
