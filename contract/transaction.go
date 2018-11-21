package sc

/*
#include "v8/engine.h"
*/
import "C"
import (
	"encoding/hex"
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

//export TransactionGetFunc
func TransactionGetFunc(address unsafe.Pointer, context unsafe.Pointer) {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Failed to get V8 engine!")
		return
	}

	if cb == nil || context == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart Contract: Invalid get transaction params!")
		return
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return
	}

	tx := C.struct_transaction_t{}
	tx.id = C.CString(hex.EncodeToString(engine.tx.ID))
	tx.tip = C.ulonglong(engine.tx.Tip)

	tx.vin_length = C.int(len(engine.tx.Vin))
	vins := make([]C.struct_transaction_vin_t, len(engine.tx.Vin))
	for _, txVin := range engine.tx.Vin {
		vin := C.struct_transaction_vin_t{}
		vin.txid = C.CString(hex.EncodeToString(txVin.Txid))
		vin.vout = C.int(txVin.Vout)
		vin.signature = C.CString(hex.EncodeToString(txVin.Signature))
		vin.pubkey = C.CString(hex.EncodeToString(txVin.PubKey))
		vins = append(vins, vin)
	}
	tx.vin = (*C.struct_transaction_vin_t)(unsafe.Pointer(&vins[0]))

	tx.vout_length = C.int(len(engine.tx.Vout))
	vouts := make([]C.struct_transaction_vout_t, len(engine.tx.Vout))
	for _, txVout := range engine.tx.Vout {
		vout := C.struct_transaction_vout_t{}
		vout.amount = C.longlong(txVout.Value.Int64())
		vout.pubkeyhash = C.CString(hex.EncodeToString(txVout.PubKeyHash.PubKeyHash))
		vouts = append(vouts, vout)
	}
	tx.vout = (*C.struct_transaction_vout_t)(unsafe.Pointer(&vouts[0]))

	C.SetTransactionData((*C.struct_transaction_t)(unsafe.Pointer(&tx)), context)
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
		return 0
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return 0
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

	if int(index) >= len(engine.tx.Vin) {
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
		return 0
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return 0
	}

	if int(index) >= len(engine.tx.Vin) {
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

	if int(index) >= len(engine.tx.Vin) {
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

	if int(index) >= len(engine.tx.Vin) {
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
		return 0
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return 0
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
		return 0
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get transaction in v8 engine")
		return 0
	}

	if int(index) >= len(engine.tx.Vout) {
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

	if int(index) >= len(engine.tx.Vout) {
		logger.WithFields(logger.Fields{
			"contractTransactionId": hex.EncodeToString(engine.tx.ID),
		}).Debug("Smart contract: vout index overflow")
	}

	return C.CString(hex.EncodeToString(engine.tx.Vout[index].PubKeyHash.PubKeyHash))
}
