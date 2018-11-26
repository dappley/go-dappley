package vm

/*
#include "v8/engine.h"
*/
import "C"
import (
	"encoding/hex"
	"unsafe"

	logger "github.com/sirupsen/logrus"
)

//export PrevUtxoGetFunc
func PrevUtxoGetFunc(address unsafe.Pointer, context unsafe.Pointer) {
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
		}).Debug("Smart Contract: Invalid get utxo params!")
		return
	}

	if engine.prevUtxos == nil {
		logger.WithFields(logger.Fields{
			"contractAddr": addr,
		}).Debug("Smart contract: Failed to get prevUtxo in v8 engine")
		return
	}

	utxoLength := C.int(len(engine.prevUtxos))
	utxosAddr := (*C.struct_utxo_t)(C.malloc(C.ulong(C.sizeof_struct_utxo_t * utxoLength)))
	defer C.free(unsafe.Pointer(utxosAddr))
	utxos := (*[1 << 30]C.struct_utxo_t)(unsafe.Pointer(utxosAddr))[:utxoLength:utxoLength]
	for index, prevUtxo := range engine.prevUtxos {
		utxo := &utxos[index]
		utxo.txid = C.CString(hex.EncodeToString(prevUtxo.Txid))
		defer C.free(unsafe.Pointer(utxo.txid))

		utxo.tx_index = C.int(prevUtxo.TxIndex)

		utxo.value = C.longlong(prevUtxo.Value.Int64())
		utxo.pubkeyhash = C.CString(hex.EncodeToString(prevUtxo.PubKeyHash.PubKeyHash))
		defer C.free(unsafe.Pointer(utxo.pubkeyhash))

		utxo.address = C.CString(prevUtxo.PubKeyHash.GenerateAddress().Address)
		defer C.free(unsafe.Pointer(utxo.address))
	}

	C.SetPrevUtxoData(utxosAddr, utxoLength, context)
}
