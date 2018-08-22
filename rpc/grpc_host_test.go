package rpc

import (
	"testing"
	"github.com/dappley/go-dappley/network"
	"google.golang.org/grpc"
	"github.com/dappley/go-dappley/rpc/pb"
	"golang.org/x/net/context"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestNewGrpcServer(t *testing.T) {
	node := network.NewNode(nil)
	grpcServer := NewGrpcServer(node)
	assert.Equal(t,node,grpcServer.node)
}

//integration test
func TestServer_StartRPC(t *testing.T) {

	pid := "QmWsMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	addr := "/ip4/127.0.0.1/tcp/10000"
	node := network.FakeNodeWithPeer(pid, addr)
	//start grpc server
	grpcServer := NewGrpcServer(node)
	go StartRpc(grpcServer)

	time.Sleep(time.Millisecond*100)
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(port, grpc.WithInsecure())
	assert.Nil(t, err)
	defer conn.Close()

	c := rpcpb.NewConnectClient(conn)
	response, err := c.RpcGetPeerInfo(context.Background(),&rpcpb.GetPeerInfoRequest{})
	assert.Nil(t, err)

	ret := &network.PeerList{}
	ret.FromProto(response.PeerList)
	assert.Equal(t,node.GetPeerList(),ret)

}