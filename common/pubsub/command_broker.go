package pubsub

import (
	"errors"
)

var (
	ErrTopicOccupied   = errors.New("Topic already occupied")
	ErrNoHandlersFound = errors.New("No command handlers")
)

type commandHandler func(input interface{})

type CommandBroker struct {
	reservedTopics map[string]bool
	handlers       map[string][]commandHandler
}

//NewCommandBroker creates a commandBroker instance
func NewCommandBroker(reservedTopcis []string) *CommandBroker {

	cb := &CommandBroker{
		reservedTopics: make(map[string]bool),
		handlers:       make(map[string][]commandHandler, 0),
	}

	for _, topic := range reservedTopcis {
		cb.reservedTopics[topic] = true
	}

	return cb
}

//subscribeCmd adds a handler to the topic "command"
func (cb *CommandBroker) Subscribe(command string, handler commandHandler) error {
	_, isReservedTopic := cb.reservedTopics[command]

	if _, isTopicOccupied := cb.handlers[command]; isTopicOccupied && !isReservedTopic {
		return ErrTopicOccupied
	}

	cb.handlers[command] = append(cb.handlers[command], handler)
	return nil
}

//Dispatch publishes a topic and run the topic handler
func (cb *CommandBroker) Dispatch(command string, content interface{}) error {
	if _, ok := cb.handlers[command]; !ok {
		return ErrNoHandlersFound
	}

	for _, handler := range cb.handlers[command] {
		if handler != nil {
			go handler(content)
		}
	}
	return nil
}
