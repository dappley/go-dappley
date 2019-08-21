package vm

import "C"
import (
	"unsafe"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

//VerifyAddressFunc verify address is valid
//export VerifyAddressFunc
func VerifyAddressFunc(address *C.char, gasCnt *C.size_t) bool {
	// calculate Gas.
	*gasCnt = C.size_t(VerifyAddressGasBase)
	addr := account.NewAddress(C.GoString(address))
	return addr.IsValid()
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

//export DeleteContractFunc
func DeleteContractFunc(handler unsafe.Pointer) int {
	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.WithFields(logger.Fields{
			"handler":  uint64(uintptr(handler)),
			"function": "Blockchain.DeleteContractFunc",
		}).Debug("SmartContract: failed to get the engine instance!")
		return 1
	}

	contractAddr := engine.contractAddr
	utxos := engine.contractUTXOs
	createUtxo := engine.contractCreateUTXO
	utxos = append(utxos, createUtxo)
	sourceTXID := engine.sourceTXID
	if !contractAddr.IsValid() {
		return 1
	}

	transferTX := core.NewSmartContractDestoryTX(utxos, contractAddr, sourceTXID)

	engine.generatedTXs = append(
		engine.generatedTXs,
		&transferTX,
	)

	return 0

}

//TransferFunc transfer amount from contract to address
//export TransferFunc
func TransferFunc(handler unsafe.Pointer, to *C.char, amount *C.char, tip *C.char, gasCnt *C.size_t) int {
	toAddr := account.NewAddress(C.GoString(to))
	amountValue, err := common.NewAmountFromString(C.GoString(amount))
	if err != nil {
		logger.Warn("SmartContract: transfer amount is invalid!")
		return 1
	}
	tipValue, err := common.NewAmountFromString(C.GoString(tip))
	if err != nil {
		logger.Warn("SmartContract: tip for the transfer is invalid!")
		return 1
	}

	engine := getV8EngineByAddress(uint64(uintptr(handler)))
	if engine == nil {
		logger.Warn("SmartContract: failed to get the engine instance!")
		return 1
	}

	// calculate Gas.
	*gasCnt = C.size_t(TransferGasBase)

	contractAddr := engine.contractAddr
	utxos := engine.contractUTXOs
	sourceTXID := engine.sourceTXID
	if !contractAddr.IsValid() {
		return 1
	}

	utxosToSpend, ok := prepareUTXOs(utxos, amountValue.Add(tipValue))
	if !ok {
		logger.Warn("SmartContract: there is insufficient fund for the transfer!")
		return 1
	}

	transferTX, err := core.NewContractTransferTX(utxosToSpend, contractAddr, toAddr, amountValue, tipValue, common.NewAmount(0), common.NewAmount(0), sourceTXID)

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
