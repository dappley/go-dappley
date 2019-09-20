package vm

import (
	"errors"
)

// Error Types
var (
	ErrExecutionTimeout               = errors.New("execution timeout")
	ErrInsufficientGas                = errors.New("insufficient gas")
	ErrExceedMemoryLimits             = errors.New("exceed memory limits")
	ErrInjectTracingInstructionFailed = errors.New("inject tracing instructions failed")
	ErrLimitHasEmpty                  = errors.New("limit args has empty")
	ErrSetMemorySmall                 = errors.New("set memory small than v8 limit")

	// vm error
	ErrExecutionFailed = errors.New("execution failed")
	ErrUnexpected      = errors.New("Unexpected sys error")
	// multi vm error
	ErrInnerExecutionFailed = errors.New("multi execution failed")
)

// define gas consume
const (
	//In blockChain
	TransferGasBase      = 2000
	VerifyAddressGasBase = 100
)

// Default gas count
var (
	// DefaultLimitsOfTotalMemorySize default limits of total memory size
	DefaultLimitsOfTotalMemorySize uint64 = 40 * 1000 * 1000
	// DefaultLimitsOfGas default limits of gas used
	DefaultLimitsOfGas uint64 = 40 * 1000 * 1000
)
