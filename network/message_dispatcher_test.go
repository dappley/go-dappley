package network

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageDispatcher_Subscribe(t *testing.T) {
	md := NewMessageDispatcher()
	var dispatcher chan []byte
	cmd := "testCmd"
	err := md.Subscribe(cmd, dispatcher)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(md.subscribers))

	err = md.Subscribe(cmd, dispatcher)
	assert.Equal(t, ErrTopicOccupied, err)
	assert.Equal(t, 1, len(md.subscribers))
}

func TestMessageDispatcher_Dispatch(t *testing.T) {
	md := NewMessageDispatcher()
	dispatcher := make(chan []byte, 0)
	cmd := "testCmd"
	data := []byte("test")

	md.Subscribe(cmd, dispatcher)
	err := md.Dispatch(cmd, data)
	assert.Equal(t, err, ErrDispatcherFull)

	dispatcher = make(chan []byte, 10)
	cmd = "testCmd2"
	md.Subscribe(cmd, dispatcher)
	err = md.Dispatch(cmd, data)
	assert.Nil(t, err)

	rcvedData := <-dispatcher

	assert.Equal(t, data, rcvedData)
}
