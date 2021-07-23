package vm

import (
	"encoding/hex"

	"github.com/dappley/go-dappley/core/scState"
	errorValues "github.com/dappley/go-dappley/errors"

	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/storage"

	logger "github.com/sirupsen/logrus"
)

// EstimateGas returns estimated gas value of contract deploy and execution.
func EstimateGas(tx *transaction.Transaction, tailBlk *block.Block, utxoCache *utxo.UTXOCache, db storage.Storage) (uint64, error) {
	utxoIndex := lutxo.NewUTXOIndex(utxoCache)
	engine := NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	ctx := ltransaction.NewTxContract(tx)
	if ctx == nil {
		return 0, errorValues.ErrTransactionVerifyFailed
	}
	prevUtxos, err := lutxo.FindVinUtxosInUtxoPool(utxoIndex, ctx.Transaction)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"txid": hex.EncodeToString(ctx.ID),
		}).Warn("Transaction: cannot find vin while executing smart contract")
		return 0, err
	}
	isContractDeployed := ctx.IsContractDeployed(utxoIndex)
	contractState := scState.NewScState(utxoCache)
	gasCount, _, err := ctx.Execute(prevUtxos, isContractDeployed, utxoIndex, contractState, rewards, engine, tailBlk.GetHeight()+1, tailBlk)
	return gasCount, err
}
