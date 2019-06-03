package vm

import (
	"errors"
)

// Error Types
var (
	ErrEngineRepeatedStart      = errors.New("engine repeated start")
	ErrEngineNotStart           = errors.New("engine not start")
	ErrContextConstructArrEmpty = errors.New("context construct err by args empty")
	ErrEngineNotFound           = errors.New("Failed to get engine")

	ErrDisallowCallPrivateFunction     = errors.New("disallow call private function")
	ErrExecutionTimeout                = errors.New("execution timeout")
	ErrInsufficientGas                 = errors.New("insufficient gas")
	ErrExceedMemoryLimits              = errors.New("exceed memory limits")
	ErrInjectTracingInstructionFailed  = errors.New("inject tracing instructions failed")
	ErrTranspileTypeScriptFailed       = errors.New("transpile TypeScript failed")
	ErrUnsupportedSourceType           = errors.New("unsupported source type")
	ErrArgumentsFormat                 = errors.New("arguments format error")
	ErrLimitHasEmpty                   = errors.New("limit args has empty")
	ErrSetMemorySmall                  = errors.New("set memory small than v8 limit")
	ErrDisallowCallNotStandardFunction = errors.New("disallow call not standard function")

	ErrMaxInnerContractLevelLimit = errors.New("out of limit vm count")
	ErrInnerTransferFailed        = errors.New("inner transfer failed")
	ErrInnerInsufficientGas       = errors.New("preparation inner vm insufficient gas")
	ErrInnerInsufficientMem       = errors.New("preparation inner vm insufficient mem")

	ErrOutOfVmMaxGasLimit = errors.New("out of vm max gas limit")

	// vm error
	ErrExecutionFailed = errors.New("execution failed")
	ErrUnexpected      = errors.New("Unexpected sys error")
	// multi vm error
	ErrInnerExecutionFailed = errors.New("multi execution failed")
)

//define
const (
	EventNameSpaceContract    = "chain.contract" //ToRefine: move to core
	InnerTransactionErrPrefix = "inner transation err ["
	InnerTransactionResult    = "] result ["
	InnerTransactionErrEnding = "] engine index:%v"
)

//transfer err code enum
const (
	SuccessTransferFunc = iota
	SuccessTransfer
	ErrTransferGetEngine
	ErrTransferAddressParse
	ErrTransferGetAccount
	ErrTransferStringToUint128
	ErrTransferSubBalance
	ErrTransferAddBalance
	ErrTransferRecordEvent
	ErrTransferAddress
)

//the max recent block number can query
const (
	maxQueryBlockInfoValidTime = 30
	maxBlockOffset             = maxQueryBlockInfoValidTime * 24 * 3600 * 1000 / 15000 //TODO:dpos.BlockIntervalInMs
)

// define gas consume
const (
	// crypto
	CryptoSha256GasBase         = 20000
	CryptoSha3256GasBase        = 20000
	CryptoRipemd160GasBase      = 20000
	CryptoRecoverAddressGasBase = 100000
	CryptoMd5GasBase            = 6000
	CryptoBase64GasBase         = 3000

	//In blockChain
	GetTxByHashGasBase     = 1000
	GetAccountStateGasBase = 2000
	TransferGasBase        = 2000
	VerifyAddressGasBase   = 100
	GetPreBlockHashGasBase = 2000
	GetPreBlockSeedGasBase = 2000

	//inner nvm
	GetContractSourceGasBase = 5000
	InnerContractGasBase     = 32000

	//random
	GetTxRandomGasBase = 1000

	//nr
	GetLatestNebulasRankGasBase        = 20000
	GetLatestNebulasRankSummaryGasBase = 20000
)

//inner nvm
const (
	MaxInnerContractLevel = 3
)

// Default gas count
var (
	// DefaultLimitsOfTotalMemorySize default limits of total memory size
	DefaultLimitsOfTotalMemorySize uint64 = 40 * 1000 * 1000
)

