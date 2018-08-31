package rpc

import (
	"context"

	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/network"
)

type RpcService struct{
	node *network.Node
}

// SayHello implements helloworld.GreeterServer
func (rpcSerivce *RpcService) RpcCreateWallet(ctx context.Context, in *rpcpb.CreateWalletRequest) (*rpcpb.CreateWalletResponse, error) {
	return &rpcpb.CreateWalletResponse{Message: "Hello " + in.Name}, nil
}

func (rpcSerivce *RpcService) RpcGetBalance(ctx context.Context, in *rpcpb.GetBalanceRequest) (*rpcpb.GetBalanceResponse, error) {
	return &rpcpb.GetBalanceResponse{Message: "Hello " + in.Name}, nil
}

func (rpcSerivce *RpcService) RpcSend(ctx context.Context, in *rpcpb.SendRequest) (*rpcpb.SendResponse, error) {
	return &rpcpb.SendResponse{Message: "Hello " + in.Name}, nil
}

func (rpcSerivce *RpcService) RpcGetPeerInfo(ctx context.Context, in *rpcpb.GetPeerInfoRequest) (*rpcpb.GetPeerInfoResponse, error) {
	return &rpcpb.GetPeerInfoResponse{
		PeerList: rpcSerivce.node.GetPeerList().ToProto().(*networkpb.Peerlist),
	}, nil
}

func (rpcSerivce *RpcService) RpcGetBlockchainInfo(ctx context.Context, in *rpcpb.GetBlockchainInfoRequest) (*rpcpb.GetBlockchainInfoResponse, error){
	return &rpcpb.GetBlockchainInfoResponse{
		TailBlockHash: rpcSerivce.node.GetBlockchain().GetTailBlockHash(),
		BlockHeight:   rpcSerivce.node.GetBlockchain().GetMaxHeight(),
	}, nil
}
