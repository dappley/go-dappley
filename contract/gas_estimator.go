package vm

import "github.com/dappley/go-dappley/core"

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
	gasCount, _, err := ctx.Execute(*utxoIndex, scStorage, rewards, engine, parentBlock.GetHeight()+1, parentBlock)
	return gasCount, err
}
