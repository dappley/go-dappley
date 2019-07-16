package network

import (
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
			var dispatcher chan *DappRcvdCmdContext
			err := md.Subscribe(tt.cmd1, dispatcher)

			assert.Equal(t, tt.expectedErr1, err)
			err = md.Subscribe(tt.cmd2, dispatcher)
			assert.Equal(t, tt.expectedErr2, err)
			assert.Equal(t, tt.expectedNumOfSubscribers, len(md.subscribers))
		})
	}
}

func TestCommandBroker_Dispatch(t *testing.T) {
	tests := []struct {
		name             string
		cmd              string
		dispatcherChSize int
		expectedErr      error
	}{
		{
			name:             "normal case",
			cmd:              "cmd",
			dispatcherChSize: 10,
			expectedErr:      nil,
		},
		{
			name:             "Dispatch to uninitialized channel",
			cmd:              "cmd",
			dispatcherChSize: 0,
			expectedErr:      ErrDispatcherFull,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewCommandBroker(reservedTopics)
			dispatcherCh := make(chan *DappRcvdCmdContext, tt.dispatcherChSize)

			dappCmd := NewDapCmd("testCmd", []byte("test"), false)
			var source peer.ID
			rcvdCmd := NewDappRcvdCmdContext(dappCmd, source)

			md.Subscribe(rcvdCmd.GetCommandName(), dispatcherCh)
			err := md.Dispatch(rcvdCmd)
			assert.Equal(t, tt.expectedErr, err)
			if err == nil {
				rcvedData := <-dispatcherCh
				assert.Equal(t, rcvdCmd, rcvedData)
			}

		})
	}
}

func TestCommandBroker_DispatchMultiple(t *testing.T) {
	md := NewCommandBroker(reservedTopics)
	dispatcherCh1 := make(chan *DappRcvdCmdContext, 10)
	dispatcherCh2 := make(chan *DappRcvdCmdContext, 10)

	dappCmd := NewDapCmd(reservedTopics[0], []byte("test"), false)
	var source peer.ID
	rcvdCmd := NewDappRcvdCmdContext(dappCmd, source)

	md.Subscribe(rcvdCmd.GetCommandName(), dispatcherCh1)
	md.Subscribe(rcvdCmd.GetCommandName(), dispatcherCh2)
	err := md.Dispatch(rcvdCmd)
	assert.Nil(t, err)

	rcvedData := <-dispatcherCh1
	assert.Equal(t, rcvdCmd, rcvedData)
	rcvedData = <-dispatcherCh2
	assert.Equal(t, rcvdCmd, rcvedData)

}
