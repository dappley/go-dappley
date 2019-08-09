package vm

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/logic/transaction_logic"
	"github.com/dappley/go-dappley/logic/utxo_logic"

	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

// EstimateGas returns estimated gas value of contract deploy and execution.
func EstimateGas(bc *blockchain_logic.Blockchain, tx *transaction.Transaction) (uint64, error) {
	parentBlock, _ := bc.GetTailBlock()
	utxoIndex := utxo_logic.NewUTXOIndex(bc.GetUtxoCache())
	scStorage := core.LoadScStateFromDatabase(bc.GetDb())
	engine := NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	ctx := tx.ToContractTx()
	if ctx == nil {
		return 0, blockchain_logic.ErrTransactionVerifyFailed
	}
	prevUtxos, err := utxo_logic.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"txid": hex.EncodeToString(ctx.ID),
		}).Warn("Transaction: cannot find vin while executing smart contract")
		return 0, err
	}
	isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0
	gasCount, _, err := transaction_logic.Execute(ctx, prevUtxos, isSCUTXO, *utxoIndex, scStorage, rewards, engine, parentBlock.GetHeight()+1, parentBlock)
	return gasCount, err
}
