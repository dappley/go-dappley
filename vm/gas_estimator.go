package vm

import (
	"encoding/hex"

	"github.com/dappley/go-dappley/core/scState"

	"errors"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/storage"

	logger "github.com/sirupsen/logrus"
)

var (
	ErrTransactionVerifyFailed = errors.New("transaction verification failed")
)

// EstimateGas returns estimated gas value of contract deploy and execution.
func EstimateGas(tx *transaction.Transaction, tailBlk *block.Block, utxoCache *utxo.UTXOCache, db storage.Storage) (uint64, error) {
	utxoIndex := lutxo.NewUTXOIndex(utxoCache)
	scStorage := scState.LoadScStateFromDatabase(db)
	engine := NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	ctx := tx.ToContractTx()
	if ctx == nil {
		return 0, ErrTransactionVerifyFailed
	}
	prevUtxos, err := lutxo.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"txid": hex.EncodeToString(ctx.ID),
		}).Warn("Transaction: cannot find vin while executing smart contract")
		return 0, err
	}
	isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0
	gasCount, _, err := ltransaction.Execute(ctx, prevUtxos, isSCUTXO, *utxoIndex, scStorage, rewards, engine, tailBlk.GetHeight()+1, tailBlk)
	return gasCount, err
}
