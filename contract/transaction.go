package sc

//#include "v8/lib/transaction_struct.h"
import "C"
import (
	"encoding/hex"
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

//export TransactionGetFunc
func TransactionGetFunc(address unsafe.Pointer) *C.struct_transaction_t {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	tx := C.struct_transaction_t{}
	tx._id = C.CString(hex.EncodeToString(engine.tx.ID))
	tx._tip = C.longlong(engine.tx.Tip)

	tx._vin_length = C.int(len(engine.tx.Vin))
	vins := make([]C.struct_transaction_vin_t, len(engine.tx.Vin))
	for i, txVin := range engine.tx.Vin {
		vin := C.struct_transaction_vin_t{}
		vin._txid = C.CString(hex.EncodeToString(txVin.Txid))
		vin._vout = C.int(txVin.Vout)
		vin._signature = C.CString(hex.EncodeToString(txVin.Signature))
		vin._pubkey = C.CString(hex.EncodeToString(txVin.PubKey))
		append(vins, vin)
	}
	tx._vin = unsafe.Pointer(&vins[0])

	tx._vout_length = C.int(len(engine.tx.Vout))
	vouts := make([]C.struct_transaction_vout_t, len(engine.tx.Vout))
	for i, txVout := range engine.tx.Vout {
		vout := C.struct_transaction_vout_t{}
		vout._amount = C.longlong(txVout.Value.Int64())
		vout._pubkeyhash = C.CString(hex.EncodeToString(txVout.PubKeyHash))
		append(vouts, vout)
	}
	tx._vout = unsafe.Pointer(&vouts[0])

	return unsafe.Pointer(&tx)
}

//export TransactionGetIdFunc
func TransactionGetIdFunc(address unsafe.Pointer) *C.char {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	return C.CString(hex.EncodeToString(engine.tx.ID))
}

//export TransactionGetVinLength
func TransactionGetVinLength(address unsafe.Pointer) C.int {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}
	return C.int(len(engine.tx.Vin))
}

//export TransactionGetVinTxidFunc
func TransactionGetVinTxidFunc(address unsafe.Pointer, index C.int) *C.char {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	if index >= len(engine.tx.Vin) {
		logger.WithFields(logger.Fields{
			"contractTransactionId": hex.EncodeToString(engine.tx.ID),
		}).Debug("Smart contract: vin index overflow")
	}

	return C.CString(hex.EncodeToString(engine.tx.Vin[int(index)].Txid))
}

//export TransactionGetVinVoutFunc
func TransactionGetVinVoutFunc(address unsafe.Pointer, index C.int) C.int {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	if index >= len(engine.tx.Vin) {
		logger.WithFields(logger.Fields{
			"contractTransactionId": hex.EncodeToString(engine.tx.ID),
		}).Debug("Smart contract: vin index overflow")
	}

	return C.int(engine.tx.Vin[int(index)].Vout)
}

//export TransactionGetVinSignatureFunc
func TransactionGetVinSignatureFunc(address unsafe.Pointer, index C.int) *C.char {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	if index >= len(engine.tx.Vin) {
		logger.WithFields(logger.Fields{
			"contractTransactionId": hex.EncodeToString(engine.tx.ID),
		}).Debug("Smart contract: vin index overflow")
	}

	return C.CString(hex.EncodeToString(engine.tx.Vin[int(index)].Signature))
}

//export TransactionGetVinPubkeyFunc
func TransactionGetVinPubkeyFunc(address unsafe.Pointer, index C.int) *C.char {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	if index >= len(engine.tx.Vin) {
		logger.WithFields(logger.Fields{
			"contractTransactionId": hex.EncodeToString(engine.tx.ID),
		}).Debug("Smart contract: vin index overflow")
	}

	return C.CString(hex.EncodeToString(engine.tx.Vin[int(index)].PubKey))
}

//export TransactionGetVoutLength
func TransactionGetVoutLength(address unsafe.Pointer) C.int {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}
	return C.int(len(engine.tx.Vout))
}

//export TransactionGetVoutAmount
func TransactionGetVoutAmount(address unsafe.Pointer, index C.int) C.longlong {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	if index >= len(engine.tx.Vout) {
		logger.WithFields(logger.Fields{
			"contractTransactionId": hex.EncodeToString(engine.tx.ID),
		}).Debug("Smart contract: vout index overflow")
	}
	return C.longlong(engine.tx.Vout[index].Value.Int64())
}

//export TransactionGetVountPubkeyHash
func TransactionGetVountPubkeyHash(address unsafe.Pointer, index C.int) *C.char {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return nil
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return nil
	}

	if index >= len(engine.tx.Vout) {
		logger.WithFields(logger.Fields{
			"contractTransactionId": hex.EncodeToString(engine.tx.ID),
		}).Debug("Smart contract: vout index overflow")
	}

	return C.CString(hex.EncodeToString(engine.tx.Vout[index].PubKeyHash.PubKeyHash))
}
