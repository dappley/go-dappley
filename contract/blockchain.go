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
	contractPubKey, ok := contractAddr.GetPubKeyHash()
	if !ok {
		return 1
	}

	// TODO: to be removed after implementing NewContractTransferTX
	var txins []core.TXInput

	// TODO: need to refactor and expose "get utxos by amount" from UTXOIndex, given a []*UTXO
	total := common.NewAmount(0)
	for _, utxo := range utxos {
		total = total.Add(utxo.Value)
		txins = append(txins, core.TXInput{
			Txid:      utxo.Txid,
			Vout:      utxo.TxIndex,
			Signature: sourceTXID,
			PubKey:    contractPubKey,
		})
		if total.Cmp(amountValue.Add(tipValue)) >= 0 {
			break
		}
	}

	change, err := total.Sub(amountValue.Add(tipValue))
	if err != nil {
		logger.WithFields(logger.Fields{
			"balance": total,
			"amount":  amountValue,
			"tip":     tipValue,
		}).Debug("Smart Contract: Insufficient balance!")
		return 1
	}
	engine.generatedTXs = []*core.Transaction{
		{
			[]byte("contractGenerated"),
			txins,
			[]core.TXOutput{
				*core.NewTxOut(amountValue, toAddr, ""),
				*core.NewTxOut(change, contractAddr, ""),
			},
			tipValue.Uint64(),
		},
	}

	//engine.generatedTXs = append(
	//	engine.generatedTXs,
	//	core.NewContractTransferTX(contractAddr, toAddr, amountValue, tipValue),
	//)

	return 0
}
