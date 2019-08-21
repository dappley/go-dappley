package core

import (
	"errors"
)

// Error Types
var (
	ErrInvalidGasPrice = errors.New("invalid gas price, should be in (0, 10^12]")
	ErrInvalidGasLimit = errors.New("invalid gas limit, should be in (0, 5*10^10]")

	ErrInsufficientBalance = errors.New("Transaction: insufficient balance, cannot pay for GasLimit")
	ErrOutOfGasLimit       = errors.New("out of gas limit")

	// vm error
	ErrExecutionFailed = errors.New("execution failed")

	ErrUnsupportedSourceType = errors.New("unsupported source type")
)
