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

//NewDappSdk creates a new DappSdkConn instance that connects to a Dappley node with grpc port
func NewDappSdk(gprcPort uint32) *DappSdkConn {
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

func (sdkc *DappSdkConn) GetRpcClient() rpcpb.RpcServiceClient { return sdkc.rpcClient }

func (sdkc *DappSdkConn) GetAdminClient() rpcpb.AdminServiceClient { return sdkc.adminClient }
