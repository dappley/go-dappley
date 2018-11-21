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

	if context == nil {
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
	vinAddr := (*C.struct_transaction_vin_t)(C.malloc(C.ulong(C.sizeof_struct_transaction_vin_t * tx.vin_length)))
	vins := (*[1 << 30]C.struct_transaction_vin_t)(unsafe.Pointer(vinAddr))[:tx.vin_length:tx.vin_length]
	for index, txVin := range engine.tx.Vin {
		vins[index].txid = C.CString(hex.EncodeToString(txVin.Txid))
		vins[index].vout = C.int(txVin.Vout)
		vins[index].signature = C.CString(hex.EncodeToString(txVin.Signature))
		vins[index].pubkey = C.CString(hex.EncodeToString(txVin.PubKey))
	}
	tx.vin = vinAddr

	tx.vout_length = C.int(len(engine.tx.Vout))
	voutAddr := (*C.struct_transaction_vout_t)(C.malloc(C.ulong(C.sizeof_struct_transaction_vout_t * tx.vout_length)))
	vouts := (*[1 << 30]C.struct_transaction_vout_t)(unsafe.Pointer(voutAddr))[:tx.vout_length:tx.vout_length]
	for index, txVout := range engine.tx.Vout {
		vouts[index].amount = C.longlong(txVout.Value.Int64())
		vouts[index].pubkeyhash = C.CString(hex.EncodeToString(txVout.PubKeyHash.PubKeyHash))
	}
	tx.vout = voutAddr

	C.SetTransactionData((*C.struct_transaction_t)(unsafe.Pointer(&tx)), context)

	for index, txVout := range engine.tx.Vout {
		C.free(unsafe.Pointer(vouts[index].pubkeyhash))
	}
	C.free(unsafe.Pointer(voutAddr))

	for index, txVin := range engine.tx.Vin {
		C.free(unsafe.Pointer(vins[index].txid))
		C.free(unsafe.Pointer(vins[index].signature))
		C.free(unsafe.Pointer(vins[index].pubkey))
	}
	C.free(unsafe.Pointer(vinAddr))

	C.free(unsafe.Pointer(tx.id))
}
