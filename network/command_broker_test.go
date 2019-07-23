package network

import (
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

			var handler network_model.CommandHandlerFunc
			err := md.Subscribe(tt.cmd1, handler)
			assert.Equal(t, tt.expectedErr1, err)
			err = md.Subscribe(tt.cmd2, handler)
			assert.Equal(t, tt.expectedErr2, err)
			assert.Equal(t, tt.expectedNumOfSubscribers, len(md.handlers))
		})
	}
}

func TestCommandBroker_Dispatch(t *testing.T) {
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

			var handler network_model.CommandHandlerFunc
			handler = func(command *network_model.DappRcvdCmdContext) {
			}
			md.Subscribe(tt.subScribedCmd, handler)

			//fake received command and then dispatch
			dappCmd := network_model.NewDappCmd(tt.dispatchedCmd, []byte("test"), false)
			var source peer.ID
			rcvdCmd := network_model.NewDappRcvdCmdContext(dappCmd, source)
			err := md.Dispatch(rcvdCmd)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestCommandBroker_DispatchMultiple(t *testing.T) {
	md := NewCommandBroker(reservedTopics)
	var handler network_model.CommandHandlerFunc
	handler = func(command *network_model.DappRcvdCmdContext) {}

	//Both handlers subscribe to reserved topic
	topic := reservedTopics[0]
	md.Subscribe(topic, handler)

	dappCmd := network_model.NewDappCmd(topic, []byte("test"), false)
	var source peer.ID
	rcvdCmd := network_model.NewDappRcvdCmdContext(dappCmd, source)

	assert.Nil(t, md.Dispatch(rcvdCmd))

}
