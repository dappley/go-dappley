package network

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}

type Subscriber interface {
	GetCommandHandler() map[string]CommandHandlerFunc
	SetCommandSendCh(commandSendCh chan *DappSendCmdContext)
}
