package pubsub

import (
	"errors"
)

var (
	ErrTopicOccupied      = errors.New("Topic already occupied")
	ErrNoSubscribersFound = errors.New("No command handlers")
)

type CommandHandler func(input interface{})

type CommandBroker struct {
	reservedTopics map[string]bool
	subscribers    map[string][]Subscriber
}

//NewCommandBroker creates a commandBroker instance
func NewCommandBroker(reservedTopic []string) *CommandBroker {

	cb := &CommandBroker{
		reservedTopics: make(map[string]bool),
		subscribers:    make(map[string][]Subscriber, 0),
	}

	for _, topic := range reservedTopic {
		cb.reservedTopics[topic] = true
	}

	return cb
}

//subscribeCmd adds a handler to the topic "command"
func (cb *CommandBroker) AddSubscriber(subscriber Subscriber) error {
	for _, topic := range subscriber.GetSubscribedTopics() {
		if cb.isTopicSubscribed(topic) && !cb.isReservedTopic(topic) {
			return ErrTopicOccupied
		}
		cb.subscribers[topic] = append(cb.subscribers[topic], subscriber)
	}
	return nil
}

func (cb *CommandBroker) isReservedTopic(topic string) bool {
	_, isReservedTopic := cb.reservedTopics[topic]
	return isReservedTopic
}

func (cb *CommandBroker) isTopicSubscribed(topic string) bool {
	_, isTopicSubscribed := cb.subscribers[topic]
	return isTopicSubscribed
}

//Dispatch publishes a topic and run the topic handler
func (cb *CommandBroker) Dispatch(topic string, content interface{}) error {
	if _, ok := cb.subscribers[topic]; !ok {
		return ErrNoSubscribersFound
	}

	for _, subscriber := range cb.subscribers[topic] {
		if subscriber != nil {
			handler := subscriber.GetCommandHandler(topic)
			if handler != nil {
				go handler(content)
			}
		}
	}
	return nil
}
