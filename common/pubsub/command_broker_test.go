package pubsub_test

import (
	"testing"

	"github.com/dappley/go-dappley/common/pubsub"
	"github.com/dappley/go-dappley/common/pubsub/mocks"
	errval "github.com/dappley/go-dappley/errors"
	"github.com/stretchr/testify/assert"
)

func TestCommandBroker_Subscribe(t *testing.T) {
	var (
		reservedTopics = []string{
			"FakeReservedTopicName",
		}
	)

	tests := []struct {
		name        string
		cmd1        string
		cmd2        string
		expectedErr error
	}{
		{
			name:        "Listen different topics",
			cmd1:        "cmd",
			cmd2:        "cmd2",
			expectedErr: nil,
		},
		{
			name:        "Listen same unreserved topics",
			cmd1:        "cmd",
			cmd2:        "cmd",
			expectedErr: errval.TopicOccupied,
		},
		{
			name:        "Listen same reserved topics",
			cmd1:        reservedTopics[0],
			cmd2:        reservedTopics[0],
			expectedErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := pubsub.NewCommandBroker(reservedTopics)

			subscriber := &mocks.Subscriber{}
			subscriber.On("GetSubscribedTopics").Return([]string{tt.cmd1, tt.cmd2})

			err := md.AddSubscriber(subscriber)
			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedErr, err)
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
			expectedErr:   errval.NoSubscribersFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := pubsub.NewCommandBroker(reservedTopics)

			var handler pubsub.TopicHandler
			handler = func(input interface{}) {
			}
			subscriber := &mocks.Subscriber{}
			subscriber.On("GetSubscribedTopics").Return([]string{tt.subScribedCmd})
			subscriber.On("GetTopicHandler", tt.dispatchedCmd).Return(handler)
			md.AddSubscriber(subscriber)

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

	md := pubsub.NewCommandBroker(reservedTopics)
	var handler pubsub.TopicHandler
	handler = func(input interface{}) {}

	//Both handlers subscribe to reserved topic
	topic := reservedTopics[0]
	subscriber := &mocks.Subscriber{}
	subscriber.On("GetSubscribedTopics").Return([]string{topic})
	subscriber.On("GetTopicHandler", topic).Return(handler)
	md.AddSubscriber(subscriber)

	var input interface{}

	assert.Nil(t, md.Dispatch(topic, input))

}
