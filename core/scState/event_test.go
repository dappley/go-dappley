package scState

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEventManager(t *testing.T) {
	em := NewEventManager()
	assert.NotNil(t, em.eventBus)
}

func TestEventManager_Trigger(t *testing.T) {
	em := NewEventManager()

	count := 0
	cb := func(s *Event) {
		count = 1
	}

	event := NewEvent("topic", "data")

	em.Subscribe(event.topic, cb)
	em.Trigger([]*Event{event})
	time.Sleep(time.Second)
	assert.Equal(t, count, 1)
}

func TestEventManager_Unsubscribe(t *testing.T) {
	em := NewEventManager()

	count := 0
	cb := func(s *Event) {
		count = 1
	}

	event := NewEvent("topic", "data")

	em.Subscribe(event.topic, cb)
	em.Unsubscribe(event.topic, cb)
	em.Trigger([]*Event{event})
	time.Sleep(time.Second)
	assert.Equal(t, count, 0)
}
