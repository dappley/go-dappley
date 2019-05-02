package sdk

import (
	"fmt"
	"github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc"
)

type DappSdkGrpcClient struct {
	adminClient rpcpb.AdminServiceClient
	rpcClient   rpcpb.RpcServiceClient
}

//NewDappSdkGrpcClient creates a new DappSdkGrpcClient instance that connects to a Dappley node with grpc port
func NewDappSdkGrpcClient(gprcPort uint32) *DappSdkGrpcClient {
	//TODO: the SDK is solely for tools now. It means you have to run the Dappley node locally to use the sdk
	conn, err := grpc.Dial(fmt.Sprint(":", gprcPort), grpc.WithInsecure())
	if err != nil {
		return nil
	}
	return &DappSdkGrpcClient{
		adminClient: rpcpb.NewAdminServiceClient(conn),
		rpcClient:   rpcpb.NewRpcServiceClient(conn),
	}
}

func (sdkClient *DappSdkGrpcClient) GetRpcClient() rpcpb.RpcServiceClient { return sdkClient.rpcClient }

func (sdkClient *DappSdkGrpcClient) GetAdminClient() rpcpb.AdminServiceClient {
	return sdkClient.adminClient
}
