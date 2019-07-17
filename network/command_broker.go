package network

import (
	"errors"
)

var (
	ErrTopicOccupied  = errors.New("Topic already occupied")
	ErrDispatcherFull = errors.New("Dispatch channel full")
)

type CommandBroker struct {
	reservedTopics map[string]bool
	subscribers    map[string][]chan *DappRcvdCmdContext
}

func NewCommandBroker(reservedTopcis []string) *CommandBroker {

	cb := &CommandBroker{
		reservedTopics: make(map[string]bool),
		subscribers:    make(map[string][]chan *DappRcvdCmdContext, 0),
	}

	for _, topic := range reservedTopcis {
		cb.reservedTopics[topic] = true
	}

	return cb
}

func (cb *CommandBroker) Subscribe(cmd string, dispatcherChan chan *DappRcvdCmdContext) error {
	_, isReservedTopic := cb.reservedTopics[cmd]

	if _, isTopicOccupied := cb.subscribers[cmd]; isTopicOccupied && !isReservedTopic {
		return ErrTopicOccupied
	}

	cb.subscribers[cmd] = append(cb.subscribers[cmd], dispatcherChan)
	return nil
}

func (cb *CommandBroker) Dispatch(cmd *DappRcvdCmdContext) error {
	if _, ok := cb.subscribers[cmd.GetCommandName()]; !ok {
		return nil
	}

	var err error

	for _, subscriber := range cb.subscribers[cmd.GetCommandName()] {
		select {
		case subscriber <- cmd:
		default:
			err = ErrDispatcherFull
		}
	}
	return err
}
