package sdk

import (
	"context"
	"github.com/dappley/go-dappley/rpc/pb"
)

type DappSdkBlockchain struct {
	conn *DappSdkConn
}

func NewDappSdkBlockchain(conn *DappSdkConn) *DappSdkBlockchain {
	return &DappSdkBlockchain{conn}
}

func (sdkb *DappSdkBlockchain) GetBlockHeight() (uint64, error) {
	resp, err := sdkb.conn.rpcClient.RpcGetBlockchainInfo(
		context.Background(),
		&rpcpb.GetBlockchainInfoRequest{},
	)

	if err != nil || resp == nil {
		return 0, err
	}

	return resp.BlockHeight, nil
}

func (sdkb *DappSdkBlockchain) GetBalance(address string) (int64, error) {
	response, err := sdkb.conn.rpcClient.RpcGetBalance(context.Background(), &rpcpb.GetBalanceRequest{Address: address})
	if err != nil {
		return 0, err
	}
	return response.Amount, err
}
