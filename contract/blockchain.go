package vm

import "C"
import (
	"unsafe"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

//VerifyAddressFunc verify address is valid
//export VerifyAddressFunc
func VerifyAddressFunc(address *C.char) bool {
	addr := core.NewAddress(C.GoString(address))
	return addr.ValidateAddress()
}

func prepareUTXOs(utxos []*core.UTXO, amount *common.Amount) ([]*core.UTXO, bool) {
	sum := common.NewAmount(0)

	if len(utxos) < 1 {
		return nil, false
	}

	for i, u := range utxos {
		sum = sum.Add(u.Value)
		if sum.Cmp(amount) >= 0 {
			return utxos[:i+1], true
		}
	}
	return nil, false
}

//TransferFunc transfer amount from contract to address
//export TransferFunc
func TransferFunc(handler unsafe.Pointer, to *C.char, amount *C.char, tip *C.char) int {
	toAddr := core.NewAddress(C.GoString(to))
	amountValue, err := common.NewAmountFromString(C.GoString(amount))
	if err != nil {
		logger.WithFields(logger.Fields{
			"amount": amount,
		}).Debug("SmartContract: transfer amount is invalid!")
		return 1
	}
	tipValue, err := common.NewAmountFromString(C.GoString(tip))
	if err != nil {
		logger.WithFields(logger.Fields{
			"tip": tip,
		}).Debug("SmartContract: tip for the transfer is invalid!")
		return 1
	}

	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.WithFields(logger.Fields{
			"handler":  uint64(uintptr(handler)),
			"function": "Blockchain.TransferFunc",
			"to":       toAddr,
			"amount":   amountValue,
			"tip":      tipValue,
		}).Debug("SmartContract: failed to get the engine instance!")
		return 1
	}

	contractAddr := engine.contractAddr
	utxos := engine.contractUTXOs
	sourceTXID := engine.sourceTXID
	if !contractAddr.ValidateAddress() {
		return 1
	}

	utxosToSpend, ok := prepareUTXOs(utxos, amountValue.Add(tipValue))
	if !ok {
		logger.WithFields(logger.Fields{
			"all_utxos":      utxos,
			"spending_utxos": utxosToSpend,
			"to":             toAddr,
			"amount":         amountValue,
			"tip":            tipValue,
		}).Warn("SmartContract: there is insufficient fund for the transfer!")
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
			"function": "Blockchain.GetCurrBlockHeightFunc",
		}).Debug("SmartContract: failed to get the engine instance!")
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
			"function": "Blockchain.GetNodeAddressFunc",
		}).Debug("SmartContract: failed to get the engine instance!")
		return nil
	}

	return C.CString(engine.nodeAddr.String())
}
