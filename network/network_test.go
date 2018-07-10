package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
	"time"
)

func TestNetwork(t *testing.T){
	net1 := NewNetwork()
	host, addr, err := CreateBasicHost(10000)
	assert.Nil(t, err)
	err = net1.Setup(host,nil)
	assert.Nil(t, err)

	net2 := NewNetwork()
	host2, _, err := CreateBasicHost(10001)
	assert.Nil(t, err)
	err = net2.Setup(host2,addr)
	assert.Nil(t, err)

	b2 := core.GenerateMockBlock()
	net2.Write(b2)

	time.Sleep(time.Second)

	b1:=net1.GetBlocks()
	assert.Equal(t,*b2,*b1[0])
}