package scState

import (
	"github.com/asaskevich/EventBus"
)

type EventManager struct {
	eventBus EventBus.Bus
}

func NewEventManager() *EventManager {
	return &EventManager{EventBus.New()}
}

func (em *EventManager) Trigger(events []*Event) {
	for _, event := range events {
		em.eventBus.Publish(event.topic, event)
	}
}

func (em *EventManager) SubscribeMultiple(topics []string, cb interface{}) {
	for _, topic := range topics {
		em.Subscribe(topic, cb)
	}
}

func (em *EventManager) Subscribe(topic string, cb interface{}) {
	em.eventBus.SubscribeAsync(topic, cb, false)
}

func (em *EventManager) Unsubscribe(topic string, cb interface{}) {
	em.eventBus.Unsubscribe(topic, cb)
}
