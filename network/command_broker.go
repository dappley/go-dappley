package network

import (
	"errors"
	"github.com/dappley/go-dappley/network/network_model"
)

var (
	ErrTopicOccupied   = errors.New("Topic already occupied")
	ErrNoHandlersFound = errors.New("No command handlers")
)

type CommandBroker struct {
	reservedTopics map[string]bool
	handlers       map[string][]network_model.CommandHandlerFunc
}

//NewCommandBroker creates a commandBroker instance
func NewCommandBroker(reservedTopcis []string) *CommandBroker {

	cb := &CommandBroker{
		reservedTopics: make(map[string]bool),
		handlers:       make(map[string][]network_model.CommandHandlerFunc, 0),
	}

	for _, topic := range reservedTopcis {
		cb.reservedTopics[topic] = true
	}

	return cb
}

//subscribeCmd adds a handler to the topic "command"
func (cb *CommandBroker) Subscribe(command string, handler network_model.CommandHandlerFunc) error {
	_, isReservedTopic := cb.reservedTopics[command]

	if _, isTopicOccupied := cb.handlers[command]; isTopicOccupied && !isReservedTopic {
		return ErrTopicOccupied
	}

	cb.handlers[command] = append(cb.handlers[command], handler)
	return nil
}

//Dispatch publishes a topic and run the topic handler
func (cb *CommandBroker) Dispatch(cmd *network_model.DappRcvdCmdContext) error {
	if _, ok := cb.handlers[cmd.GetCommandName()]; !ok {
		return ErrNoHandlersFound
	}

	for _, handler := range cb.handlers[cmd.GetCommandName()] {
		if handler != nil {
			go handler(cmd)
		}
	}
	return nil
}
