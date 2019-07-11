package network

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageBroker_Subscribe(t *testing.T) {
	md := NewCommandBroker()
	var dispatcher chan *DappRcvdCmdContext
	cmd := "testCmd"
	err := md.Subscribe(cmd, dispatcher)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(md.subscribers))

	err = md.Subscribe(cmd, dispatcher)
	assert.Equal(t, ErrTopicOccupied, err)
	assert.Equal(t, 1, len(md.subscribers))
}

func TestMessageBroker_Dispatch(t *testing.T) {
	md := NewCommandBroker()
	broker := make(chan *DappRcvdCmdContext, 0)

	dappCmd := NewDapCmd("testCmd", []byte("test"), false)
	var source peer.ID
	rcvdCmd := NewDappRcvdCmdContext(dappCmd, source)

	md.Subscribe(rcvdCmd.GetCommandName(), broker)
	err := md.Dispatch(rcvdCmd)
	assert.Equal(t, err, ErrDispatcherFull)

	broker = make(chan *DappRcvdCmdContext, 10)
	rcvdCmd.command.name = "testCmd2"
	md.Subscribe(rcvdCmd.GetCommandName(), broker)
	err = md.Dispatch(rcvdCmd)
	assert.Nil(t, err)

	rcvedData := <-broker

	assert.Equal(t, rcvdCmd, rcvedData)
}
