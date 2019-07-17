package rpc

const (
	// OK success with no error
	OK uint32 = 0

	// ProtoVersionNotSupport client's proto version not support by server
	ProtoVersionNotSupport uint32 = 1

	// InvalidAddress account address is invalid
	InvalidAddress uint32 = 2

	// InvalidTransaction transaction is invalid
	InvalidTransaction uint32 = 3

	//BlockNotFound block not found in blockchain
	BlockNotFound uint32 = 4

	// GetBlocksCountOverflow get blocks count over the max num
	GetBlocksCountOverflow uint32 = 5
)
