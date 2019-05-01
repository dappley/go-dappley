package sdk

import (
	"fmt"
	"github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc"
)

type DappSdkConn struct {
	adminClient rpcpb.AdminServiceClient
	rpcClient   rpcpb.RpcServiceClient
}

//NewDappleySdk creates a new DappSdkConn instance that connects to a Dappley node with grpc port
func NewDappleySdk(gprcPort int) *DappSdkConn {
	//TODO: the SDK is solely for tools now. It means you have to run the Dappley node locally to use the sdk
	conn, err := grpc.Dial(fmt.Sprint(":", gprcPort), grpc.WithInsecure())
	if err != nil {
		return nil
	}
	return &DappSdkConn{
		adminClient: rpcpb.NewAdminServiceClient(conn),
		rpcClient:   rpcpb.NewRpcServiceClient(conn),
	}
}
