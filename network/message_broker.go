package network

import (
	"errors"
)

var (
	ErrTopicOccupied  = errors.New("Topic already occupied")
	ErrDispatcherFull = errors.New("Dispatch channel full")
)

type CommandBroker struct {
	subscribers map[string]chan *DappRcvdCmdContext
}

func NewCommandBroker() *CommandBroker {
	return &CommandBroker{
		subscribers: make(map[string]chan *DappRcvdCmdContext, 0),
	}
}

func (cb *CommandBroker) Subscribe(cmd string, dispatcherChan chan *DappRcvdCmdContext) error {
	if _, ok := cb.subscribers[cmd]; ok {
		return ErrTopicOccupied
	}

	cb.subscribers[cmd] = dispatcherChan
	return nil
}

func (cb *CommandBroker) Dispatch(cmd *DappRcvdCmdContext) error {
	if _, ok := cb.subscribers[cmd.GetCommandName()]; !ok {
		return nil
	}

	select {
	case cb.subscribers[cmd.GetCommandName()] <- cmd:
		return nil
	default:
		return ErrDispatcherFull
	}
}
