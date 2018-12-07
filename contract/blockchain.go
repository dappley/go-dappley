package vm

import "C"
import (
	"unsafe"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
)

//VerifyAddressFunc verify address is valid
//export VerifyAddressFunc
func VerifyAddressFunc(address *C.char) bool {
	addr := core.NewAddress(C.GoString(address))
	return addr.ValidateAddress()
}

//TransferFunc transfer amount from contract to address
//export TransferFunc
func TransferFunc(handler unsafe.Pointer, to *C.char, amount *C.char, tip *C.char) int {
	toAddr := core.NewAddress(C.GoString(to))
	amountValue, err := common.NewAmountFromString(C.GoString(amount))
	if err != nil {
		logger.WithFields(logger.Fields{
			"amount": amount,
		}).Debug("Smart Contract: Invalid amount used in transfer!")
		return 1
	}
	tipValue, err := common.NewAmountFromString(C.GoString(tip))
	if err != nil {
		logger.WithFields(logger.Fields{
			"tip": tip,
		}).Debug("Smart Contract: Invalid tip used in transfer!")
		return 1
	}

	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.WithFields(logger.Fields{
			"handler": uint64(uintptr(handler)),
			"to":      toAddr,
			"amount":  amountValue,
			"tip":     tipValue,
		}).Debug("Smart Contract: Failed to get the engine instance while executing transfer!")
		return 1
	}

	contractAddr := engine.contractAddr
	utxos := engine.contractUTXOs
	sourceTXID := engine.sourceTXID
	if !contractAddr.ValidateAddress() {
		return 1
	}

	utxosToSpend, ok := core.PrepareUTXOs(utxos, amountValue.Add(tipValue))
	if !ok {
		logger.WithFields(logger.Fields{
			"all utxos":      utxos,
			"spending utxos": utxosToSpend,
			"to":             toAddr,
			"amount":         amountValue,
			"tip":            tipValue,
		}).Warn("Smart Contract: Insufficient fund for the transfer!")
		return 1
	}

	transferTX, err := core.NewContractTransferTX(utxosToSpend, contractAddr, toAddr, amountValue, tipValue, sourceTXID)

	engine.generatedTXs = append(
		engine.generatedTXs,
		&transferTX,
	)

	return 0
}

//export GetCurrBlockHeightFunc
func GetCurrBlockHeightFunc(handler unsafe.Pointer) uint64 {
	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.WithFields(logger.Fields{
			"handler":  uint64(uintptr(handler)),
			"function": "Math.GetCurrBlockHeightFunc",
		}).Debug("Smart Contract: Failed to get the engine instance while executing getCurrBlockHeightFunc!")
		return 0
	}

	return engine.blkHeight
}

//export GetNodeAddressFunc
func GetNodeAddressFunc(handler unsafe.Pointer) *C.char {
	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.WithFields(logger.Fields{
			"handler":  uint64(uintptr(handler)),
			"function": "Math.GetCurrBlockHeightFunc",
		}).Debug("Smart Contract: Failed to get the engine instance while executing GetNodeAddressFunc!")
		return nil
	}

	return C.CString(engine.nodeAddr.String())
}