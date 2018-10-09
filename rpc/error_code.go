package rpc

const (
	// OK success with no error
	OK uint32 = 0

	// ProtoVersionNotSupport client's proto version not support by server
	ProtoVersionNotSupport uint32 = 1

	// InvalidAddress wallet address is invalid
	InvalidAddress uint32 = 2

	// InvalidTransaction transaction is invalid
	InvalidTransaction uint32 = 3
)
