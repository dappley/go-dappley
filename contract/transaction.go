package vm

/*
#include "v8/engine.h"
*/
import "C"
import (
	"encoding/hex"
	"math"
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

//export TransactionGetFunc
func TransactionGetFunc(address unsafe.Pointer, context unsafe.Pointer) {
	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
		}).Debug("SmartContract: failed to get V8 engine!")
		return
	}

	if context == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
		}).Debug("SmartContract: get transaction params are invalid!")
		return
	}

	if engine.tx == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
		}).Debug("SmartContract: failed to get transaction in V8 engine.")
		return
	}

	tx := C.struct_transaction_t{}
	tx.id = C.CString(hex.EncodeToString(engine.tx.ID))
	defer C.free(unsafe.Pointer(tx.id))
	tx.tip = C.CString(engine.tx.Tip.String())

	tx.vin_length = C.int(len(engine.tx.Vin))
	vinAddr := (*C.struct_transaction_vin_t)(C.malloc(C.size_t(C.sizeof_struct_transaction_vin_t * tx.vin_length)))
	defer C.free(unsafe.Pointer(vinAddr))
	var tempVin C.struct_transaction_vin_t
	vins := (*[(math.MaxInt32 - 1) / unsafe.Sizeof(tempVin)]C.struct_transaction_vin_t)(unsafe.Pointer(vinAddr))[:tx.vin_length:tx.vin_length]
	for index, txVin := range engine.tx.Vin {
		vin := &vins[index]
		vin.txid = C.CString(hex.EncodeToString(txVin.Txid))
		defer C.free(unsafe.Pointer(vin.txid))
		vin.vout = C.int(txVin.Vout)
		vin.signature = C.CString(hex.EncodeToString(txVin.Signature))
		defer C.free(unsafe.Pointer(vin.signature))
		vin.pubkey = C.CString(hex.EncodeToString(txVin.PubKey))
		defer C.free(unsafe.Pointer(vin.pubkey))
	}
	tx.vin = vinAddr

	tx.vout_length = C.int(len(engine.tx.Vout))
	voutAddr := (*C.struct_transaction_vout_t)(C.malloc(C.size_t(C.sizeof_struct_transaction_vout_t * tx.vout_length)))
	defer C.free(unsafe.Pointer(voutAddr))
	var tempVout C.struct_transaction_vout_t
	vouts := (*[(math.MaxInt32 - 1) / unsafe.Sizeof(tempVout)]C.struct_transaction_vout_t)(unsafe.Pointer(voutAddr))[:tx.vout_length:tx.vout_length]
	for index, txVout := range engine.tx.Vout {
		vout := &vouts[index]
		vout.amount = C.CString(txVout.Value.String())
		vout.pubkeyhash = C.CString(hex.EncodeToString([]byte(txVout.PubKeyHash)))
		defer C.free(unsafe.Pointer(vout.pubkeyhash))
	}
	tx.vout = voutAddr

	C.SetTransactionData((*C.struct_transaction_t)(unsafe.Pointer(&tx)), context)
}
