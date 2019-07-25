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

//export PrevUtxoGetFunc
func PrevUtxoGetFunc(address unsafe.Pointer, context unsafe.Pointer) {
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
		}).Debug("SmartContract: invalid get UTXO params!")
		return
	}

	if engine.prevUtxos == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
		}).Debug("SmartContract: failed to get prevUTXO in V8 engine")
		return
	}

	utxoLength := C.int(len(engine.prevUtxos))
	utxosAddr := (*C.struct_utxo_t)(C.malloc(C.size_t(C.sizeof_struct_utxo_t * utxoLength)))
	defer C.free(unsafe.Pointer(utxosAddr))
	var temp C.struct_utxo_t
	utxos := (*[(math.MaxInt32 - 1) / unsafe.Sizeof(temp)]C.struct_utxo_t)(unsafe.Pointer(utxosAddr))[:utxoLength:utxoLength]
	for index, prevUtxo := range engine.prevUtxos {
		utxo := &utxos[index]
		utxo.txid = C.CString(hex.EncodeToString(prevUtxo.Txid))
		defer C.free(unsafe.Pointer(utxo.txid))

		utxo.tx_index = C.int(prevUtxo.TxIndex)

		//utxo.value = C.longlong(prevUtxo.Value.Int64())
		utxo.value = C.CString(prevUtxo.Value.String())
		defer C.free(unsafe.Pointer(utxo.value))

		utxo.pubkeyhash = C.CString(prevUtxo.PubKeyHash.String())
		defer C.free(unsafe.Pointer(utxo.pubkeyhash))

		utxo.address = C.CString(prevUtxo.PubKeyHash.GenerateAddress().String())
		defer C.free(unsafe.Pointer(utxo.address))
	}

	C.SetPrevUtxoData(utxosAddr, utxoLength, context)
}
