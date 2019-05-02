package sdk

import (
	"context"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/rpc/pb"
	logger "github.com/sirupsen/logrus"
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

func (sdktx *DappSdkBlockchain) SendBatchTransactions(txs []*corepb.Transaction) error {
	_, err := sdktx.conn.rpcClient.RpcSendBatchTransaction(
		context.Background(),
		&rpcpb.SendBatchTransactionRequest{
			Transactions: txs,
		},
	)

	if err != nil {
		logger.WithError(err).Error("Unable to send batch transactions!")
		return err
	}

	logger.WithFields(logger.Fields{
		"num_of_txs": len(txs),
	}).Info("Batch Transactions are sent!")

	return nil
}
