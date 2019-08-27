package pubsub

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCommandBroker_Subscribe(t *testing.T) {
	var (
		reservedTopics = []string{
			"FakeReservedTopicName",
		}
	)

	tests := []struct {
		name                     string
		cmd1                     string
		cmd2                     string
		expectedErr1             error
		expectedErr2             error
		expectedNumOfSubscribers int
	}{
		{
			name:                     "Listen different topics",
			cmd1:                     "cmd",
			cmd2:                     "cmd2",
			expectedErr1:             nil,
			expectedErr2:             nil,
			expectedNumOfSubscribers: 2,
		},
		{
			name:                     "Listen same unreserved topics",
			cmd1:                     "cmd",
			cmd2:                     "cmd",
			expectedErr1:             nil,
			expectedErr2:             ErrTopicOccupied,
			expectedNumOfSubscribers: 1,
		},
		{
			name:                     "Listen same reserved topics",
			cmd1:                     reservedTopics[0],
			cmd2:                     reservedTopics[0],
			expectedErr1:             nil,
			expectedErr2:             nil,
			expectedNumOfSubscribers: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewCommandBroker(reservedTopics)

			var handler commandHandler
			err := md.Subscribe(tt.cmd1, handler)
			assert.Equal(t, tt.expectedErr1, err)
			err = md.Subscribe(tt.cmd2, handler)
			assert.Equal(t, tt.expectedErr2, err)
			assert.Equal(t, tt.expectedNumOfSubscribers, len(md.handlers))
		})
	}
}

func TestCommandBroker_Dispatch(t *testing.T) {

	var (
		reservedTopics = []string{
			"FakeReservedTopicName",
		}
	)

	tests := []struct {
		name          string
		subScribedCmd string
		dispatchedCmd string
		expectedErr   error
	}{
		{
			name:          "normal case",
			subScribedCmd: "cmd",
			dispatchedCmd: "cmd",
			expectedErr:   nil,
		},
		{
			name:          "unsubscribed cmd",
			subScribedCmd: "cmd",
			dispatchedCmd: "cmd1",
			expectedErr:   ErrNoHandlersFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewCommandBroker(reservedTopics)

			var handler commandHandler
			handler = func(input interface{}) {
			}
			md.Subscribe(tt.subScribedCmd, handler)

			//fake received command and then dispatch
			var input interface{}
			err := md.Dispatch(tt.dispatchedCmd, input)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestCommandBroker_DispatchMultiple(t *testing.T) {

	var (
		reservedTopics = []string{
			"FakeReservedTopicName",
		}
	)

	md := NewCommandBroker(reservedTopics)
	var handler commandHandler
	handler = func(input interface{}) {}

	//Both handlers subscribe to reserved topic
	topic := reservedTopics[0]
	md.Subscribe(topic, handler)

	var input interface{}

	assert.Nil(t, md.Dispatch(topic, input))

}
