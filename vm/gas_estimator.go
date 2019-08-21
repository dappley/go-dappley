package vm

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

// EstimateGas returns estimated gas value of contract deploy and execution.
func EstimateGas(bc *core.Blockchain, tx *core.Transaction) (uint64, error) {
	parentBlock, _ := bc.GetTailBlock()
	utxoIndex := core.NewUTXOIndex(bc.GetUtxoCache())
	scStorage := core.LoadScStateFromDatabase(bc.GetDb())
	engine := NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	ctx := tx.ToContractTx()
	if ctx == nil {
		return 0, core.ErrTransactionVerifyFailed
	}
	prevUtxos, err := ctx.FindAllTxinsInUtxoPool(*utxoIndex)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"txid": hex.EncodeToString(ctx.ID),
		}).Warn("Transaction: cannot find vin while executing smart contract")
		return 0, err
	}
	isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0
	gasCount, _, err := ctx.Execute(prevUtxos, isSCUTXO, *utxoIndex, scStorage, rewards, engine, parentBlock.GetHeight()+1, parentBlock)
	return gasCount, err
}
