package network

import (
	"errors"
)

var (
	ErrTopicOccupied  = errors.New("Topic already occupied")
	ErrDispatcherFull = errors.New("Dispatch channel full")
)

type MessageDispatcher struct {
	subscribers map[string]chan []byte
}

func NewMessageDispatcher() *MessageDispatcher {
	return &MessageDispatcher{
		subscribers: make(map[string]chan []byte, 0),
	}
}

func (md *MessageDispatcher) Subscribe(cmd string, dispatcherChan chan []byte) error {
	if _, ok := md.subscribers[cmd]; ok {
		return ErrTopicOccupied
	}

	md.subscribers[cmd] = dispatcherChan
	return nil
}

func (md *MessageDispatcher) Dispatch(cmd string, data []byte) error {
	if _, ok := md.subscribers[cmd]; !ok {
		return nil
	}

	select {
	case md.subscribers[cmd] <- data:
		return nil
	default:
		return ErrDispatcherFull
	}
}
