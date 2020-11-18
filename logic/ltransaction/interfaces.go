package ltransaction

import (
	"crypto/ecdsa"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/lutxo"
)

type ScEngineManager interface {
	CreateEngine() ScEngine
}

type ScEngine interface {
	DestroyEngine()
	ImportSourceCode(source string)
	ImportLocalStorage(state *scState.ScState)
	ImportContractAddr(contractAddr account.Address)
	ImportSourceTXID(txid []byte)
	ImportUtxoIndex(utxoIndex *lutxo.UTXOIndex)
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

// Decorator of transaction
type TxDecorator interface {
	Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO) error
	Verify(utxoIndex *lutxo.UTXOIndex, blockHeight uint64) error
}
