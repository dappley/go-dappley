// +build integration

package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
)

func TestMetricsService_RpcTransactionPoolSize(t *testing.T) {
	core.MetricsTransactionPoolSize.Clear()
	bc := core.CreateBlockchain(core.NewAddress(""), storage.NewRamStorage(), consensus.NewDPOS(), 1, nil, 100)
	metricsService := NewMetricsService(network.NewNode(bc, core.NewBlockPool(100)))

	// sanity check
	res, err := metricsService.RpcTransactionPoolSize(context.Background(), newMetricsServiceRequest())
	require.Nil(t, err)
	require.EqualValues(t, 0, res.GetSize())

	txPool := bc.GetTxPool()

	// add transaction
	tx := core.MockTransaction()
	txPool.Push(*tx)
	res, err = metricsService.RpcTransactionPoolSize(context.Background(), newMetricsServiceRequest())
	require.Nil(t, err)
	require.EqualValues(t, 1, res.GetSize())

	// exceed tx pool limit
	txPool.Push(*core.MockTransaction())
	res, err = metricsService.RpcTransactionPoolSize(context.Background(), newMetricsServiceRequest())
	require.Nil(t, err)
	require.EqualValues(t, 1, res.GetSize())

	// verify deserialization restores metric
	ramStorage := storage.NewRamStorage()
	err = txPool.SaveToDatabase(ramStorage)
	require.Nil(t, err)
	newTXPool := core.LoadTxPoolFromDatabase(ramStorage, 1)
	res, err = metricsService.RpcTransactionPoolSize(context.Background(), newMetricsServiceRequest())
	require.Nil(t, err)
	require.EqualValues(t, 1, newTXPool.GetNumOfTxInPool())
	require.EqualValues(t, 1, res.GetSize())

	// remove transaction from pool
	txPool.CleanUpMinedTxs([]*core.Transaction{tx})
	res, err = metricsService.RpcTransactionPoolSize(context.Background(), newMetricsServiceRequest())
	require.Nil(t, err)
	require.EqualValues(t, 0, res.GetSize())
}

func newMetricsServiceRequest() *rpcpb.MetricsServiceRequest {
	return &rpcpb.MetricsServiceRequest{}
}
