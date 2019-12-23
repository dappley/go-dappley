package ltransaction

import (
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/utxo"
)

type ScEngineManager interface {
	CreateEngine() ScEngine
	RunScheduledEvents(contractUtxo []*utxo.UTXO, scStorage *scState.ScState, blkHeight uint64, seed int64)
}

type ScEngine interface {
	DestroyEngine()
	ImportSourceCode(source string)
	ImportLocalStorage(state *scState.ScState)
	ImportContractAddr(contractAddr account.Address)
	ImportSourceTXID(txid []byte)
	ImportUTXOs(utxos []*utxo.UTXO)
	ImportRewardStorage(rewards map[string]string)
	ImportTransaction(tx *transaction.Transaction)
	ImportContractCreateUTXO(utxo *utxo.UTXO)
	ImportPrevUtxos(utxos []*utxo.UTXO)
	ImportCurrBlockHeight(currBlkHeight uint64)
	ImportSeed(seed int64)
	ImportNodeAddress(addr account.Address)
	GetGeneratedTXs() []*transaction.Transaction
	Execute(function, args string) (string, error)
	SetExecutionLimits(uint64, uint64) error
	ExecutionInstructions() uint64
	CheckContactSyntax(source string) error
}
