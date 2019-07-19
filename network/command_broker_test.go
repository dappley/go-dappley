package network

import (
	"github.com/dappley/go-dappley/network/mocks"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCommandBroker_Subscribe(t *testing.T) {
	tests := []struct {
		name                     string
		cmd1                     string
		cmd2                     string
		expectedErr1             error
		expectedErr2             error
		expectedNumOfSubscribers int
	}{
		{
			name:                     "Subscribe different topics",
			cmd1:                     "cmd",
			cmd2:                     "cmd2",
			expectedErr1:             nil,
			expectedErr2:             nil,
			expectedNumOfSubscribers: 2,
		},
		{
			name:                     "Subscribe same unreserved topics",
			cmd1:                     "cmd",
			cmd2:                     "cmd",
			expectedErr1:             nil,
			expectedErr2:             ErrTopicOccupied,
			expectedNumOfSubscribers: 1,
		},
		{
			name:                     "Subscribe same reserved topics",
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
			subscriber := new(mocks.Subscriber)
			subscriber.On("GetSubscribedTopics").Return([]string{tt.cmd1}).Once()
			err := md.Subscribe(subscriber)
			assert.Equal(t, tt.expectedErr1, err)

			subscriber.On("GetSubscribedTopics").Return([]string{tt.cmd2}).Once()
			err = md.Subscribe(subscriber)
			assert.Equal(t, tt.expectedErr2, err)
			assert.Equal(t, tt.expectedNumOfSubscribers, len(md.subscribers))
		})
	}
}

func TestCommandBroker_Dispatch(t *testing.T) {
	tests := []struct {
		name          string
		subScribedCmd string
		dispatchedCmd string
		expectedCb    bool
	}{
		{
			name:          "normal case",
			subScribedCmd: "cmd",
			dispatchedCmd: "cmd",
			expectedCb:    true,
		},
		{
			name:          "unsubscribed cmd",
			subScribedCmd: "cmd",
			dispatchedCmd: "cmd1",
			expectedCb:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewCommandBroker(reservedTopics)
			var handler network_model.CommandHandlerFunc
			handler = func(command *network_model.DappRcvdCmdContext) {}

			subscriber := new(mocks.Subscriber)
			subscriber.On("GetSubscribedTopics").Return([]string{tt.subScribedCmd}).Once()
			if tt.expectedCb {
				subscriber.On("GetCommandHandler", tt.subScribedCmd).Return(handler).Once()
			}

			dappCmd := network_model.NewDapCmd(tt.dispatchedCmd, []byte("test"), false)
			var source peer.ID
			rcvdCmd := network_model.NewDappRcvdCmdContext(dappCmd, source)

			md.Subscribe(subscriber)
			md.Dispatch(rcvdCmd)
		})
	}
}

func TestCommandBroker_DispatchMultiple(t *testing.T) {
	md := NewCommandBroker(reservedTopics)
	var handler network_model.CommandHandlerFunc
	handler = func(command *network_model.DappRcvdCmdContext) {}

	//Both subscribers subscribe to reserved topic
	subscriber1 := new(mocks.Subscriber)
	subscriber2 := new(mocks.Subscriber)
	topic := reservedTopics[0]

	dappCmd := network_model.NewDapCmd(topic, []byte("test"), false)
	var source peer.ID
	rcvdCmd := network_model.NewDappRcvdCmdContext(dappCmd, source)

	//both subscribers' callback function should be invoked
	subscriber1.On("GetSubscribedTopics").Return([]string{topic}).Once()
	subscriber2.On("GetSubscribedTopics").Return([]string{topic}).Once()

	subscriber1.On("GetCommandHandler", topic).Return(handler).Once()
	subscriber2.On("GetCommandHandler", topic).Return(handler).Once()

	md.Subscribe(subscriber1)
	md.Subscribe(subscriber2)

	md.Dispatch(rcvdCmd)

}
