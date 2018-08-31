package rpc

import (
	"context"

	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/rpc/pb"
)

type AdminRpcService struct{
	node *network.Node
}

func (adminRpcService *AdminRpcService) RpcAddPeer(ctx context.Context, in *rpcpb.AddPeerRequest) (*rpcpb.AddPeerResponse, error){
	status := "success"
	err:=adminRpcService.node.AddStreamByString(in.FullAddress)
	if err!=nil{
		status = err.Error()
	}
	return &rpcpb.AddPeerResponse{
		Status: status,
	}, nil
}
