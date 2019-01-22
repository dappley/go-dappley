package core

import "github.com/asaskevich/EventBus"

type Event struct{
	topic	string
	data 	string
}

func NewEvent(topic, data string) *Event{
	return &Event{topic, data}
}

type EventManager struct{
	EventBus   EventBus.Bus
}

func NewEventManager() *EventManager{
	return &EventManager{EventBus.New()}
}

func (em *EventManager) Trigger(events []*Event) {
	for _,event := range events{
		em.EventBus.Publish(event.topic, event.data)
	}
}

func (em *EventManager) Subscribe(topic string, cb interface{}){
	em.EventBus.SubscribeAsync(topic, cb, false)
}

func (em *EventManager) Unsubscribe(topic string, cb interface{}){
	em.EventBus.Unsubscribe(topic, cb)
}