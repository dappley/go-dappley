package rpc

import (
	"context"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
)

type MetricsService struct {
	node *network.Node
}

func NewMetricsService(node *network.Node) *MetricsService {
	return &MetricsService{node: node}
}

func (ms *MetricsService) RpcTransactionPoolSize(ctx context.Context, req *rpcpb.MetricsServiceRequest) (*rpcpb.TransactionPoolSizeResponse, error) {
	return &rpcpb.TransactionPoolSizeResponse{Size: core.MetricsTransactionPoolSize.Count()}, nil
}

func (ms *MetricsService) IsPrivate() bool {
	return false
}
