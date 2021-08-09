package vm

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
