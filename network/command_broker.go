package network

import (
	"errors"
	"github.com/dappley/go-dappley/network/network_model"
)

var (
	ErrTopicOccupied = errors.New("Topic already occupied")
)

type CommandBroker struct {
	reservedTopics map[string]bool
	subscribers    map[string][]Subscriber
}

func NewCommandBroker(reservedTopcis []string) *CommandBroker {

	cb := &CommandBroker{
		reservedTopics: make(map[string]bool),
		subscribers:    make(map[string][]Subscriber, 0),
	}

	for _, topic := range reservedTopcis {
		cb.reservedTopics[topic] = true
	}

	return cb
}

func (cb *CommandBroker) Subscribe(subscriber Subscriber) error {
	for _, cmd := range subscriber.GetSubscribedTopics() {
		err := cb.subscribeCmd(cmd, subscriber)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cb *CommandBroker) subscribeCmd(cmd string, subscriber Subscriber) error {
	_, isReservedTopic := cb.reservedTopics[cmd]

	if _, isTopicOccupied := cb.subscribers[cmd]; isTopicOccupied && !isReservedTopic {
		return ErrTopicOccupied
	}

	cb.subscribers[cmd] = append(cb.subscribers[cmd], subscriber)
	return nil
}

func (cb *CommandBroker) Dispatch(cmd *network_model.DappRcvdCmdContext) {
	if _, ok := cb.subscribers[cmd.GetCommandName()]; !ok {
		return
	}

	for _, subscriber := range cb.subscribers[cmd.GetCommandName()] {
		go subscriber.GetCommandHandler(cmd.GetCommandName())(cmd)
	}
}
