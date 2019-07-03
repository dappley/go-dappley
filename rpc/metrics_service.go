package rpc

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/process"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/metrics"
	metricspb "github.com/dappley/go-dappley/metrics/pb"
	"github.com/dappley/go-dappley/network"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/util"
)

type MetricsService struct {
	node *network.Node
	ds   *metrics.DataStore
	*MetricsServiceConfig
	RPCPort uint32
}

func NewMetricsService(node *network.Node, config *MetricsServiceConfig, RPCPort uint32) *MetricsService {
	return (&MetricsService{node: node, MetricsServiceConfig: config, RPCPort: RPCPort}).init()
}

func (ms *MetricsService) init() *MetricsService {
	ms.node.GetPeerManager().StartNewPingService(time.Duration(ms.PollingInterval) * time.Second)
	if ms.PollingInterval > 0 && ms.TimeSeriesInterval > 0 {
		ms.ds = metrics.NewDataStore(int(ms.TimeSeriesInterval/ms.PollingInterval), time.Duration(ms.PollingInterval)*time.Second)
		_ = ms.ds.RegisterNewMetric("dapp.cpu.percent", getCPUPercent)
		_ = ms.ds.RegisterNewMetric("dapp.txpool.size", getTransactionPoolSize)
		_ = ms.ds.RegisterNewMetric("dapp.memstats", getMemoryStats)
		_ = ms.ds.RegisterNewMetric("dapp.fork.info", ms.getNumForksInBlockChain)
		ms.ds.StartUpdate()
	}
	return ms
}

func (ms *MetricsService) RpcGetStats(ctx context.Context, request *rpcpb.MetricsServiceRequest) (*rpcpb.GetStatsResponse, error) {
	return &rpcpb.GetStatsResponse{
		Stats: &metricspb.Metrics{
			DataStore:  ms.ds.ToProto(),
			Peers:      getPeerInfo(ms.node),
			BlockStats: ms.getBlockStats(),
		},
	}, nil
}

func (ms *MetricsService) RpcGetNodeConfig(ctx context.Context, request *rpcpb.MetricsServiceRequest) (*rpcpb.GetNodeConfig, error) {
	return &rpcpb.GetNodeConfig{
		TxPoolLimit:      ms.node.GetBlockchain().GetTxPool().GetSizeLimit(),
		BlkSizeLimit:     uint32(ms.node.GetBlockchain().GetBlockSizeLimit()),
		MaxConnectionOut: uint32(ms.node.GetPeerManager().GetMaxConnectionOutCount()),
		MaxConnectionIn:  uint32(ms.node.GetPeerManager().GetMaxConnectionInCount()),
		ProducerAddress:  ms.node.GetBlockchain().GetConsensus().GetProducerAddress(),
		Producers:        ms.node.GetBlockchain().GetConsensus().GetProducers(),
		MaxProducers:     ms.getMaxProducers(),
		IPFSAddresses:    ms.node.GetIPFSAddresses(),
		RPCPort:          ms.RPCPort,
	}, nil
}

func (ms *MetricsService) getMaxProducers() uint32 {
	if dpos, ok := ms.node.GetBlockchain().GetConsensus().(*consensus.DPOS); ok {
		return uint32(dpos.GetDynasty().GetMaxProducers())
	}
	return 0
}

func (ms *MetricsService) IsPrivate() bool {
	return false
}

func getMemoryStats() metricspb.StatValue {
	stats := &runtime.MemStats{}
	runtime.ReadMemStats(stats)
	return &metricspb.Stat_MemoryStats{
		MemoryStats: &metricspb.MemoryStats{
			HeapInUse: stats.HeapInuse,
			HeapSys:   stats.HeapSys,
		},
	}
}

func getCPUPercent() metricspb.StatValue {
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		logger.Warn(err)
		return nil
	}

	percentageUsed, err := proc.CPUPercent()
	if err != nil {
		logger.Warn(err)
		return nil
	}

	return &metricspb.Stat_CpuPercentage{
		CpuPercentage: percentageUsed,
	}
}

func getTransactionPoolSize() metricspb.StatValue {
	return &metricspb.Stat_TransactionPoolSize{
		TransactionPoolSize: core.MetricsTransactionPoolSize.Count(),
	}
}

func (ms *MetricsService) getNumForksInBlockChain() metricspb.StatValue {
	numForks, longestFork := ms.node.GetBlockChainManager().NumForks()

	return &metricspb.Stat_ForkStats{
		ForkStats: &metricspb.ForkStats{
			NumForks:    numForks,
			LongestFork: longestFork,
		},
	}
}

func (ms *MetricsService) getBlockStats() []*metricspb.BlockStats {
	stats := make([]*metricspb.BlockStats, 0)
	it := ms.node.GetBlockchain().Iterator()
	cons := ms.node.GetBlockchain().GetConsensus()
	blk, err := it.Next()
	for t := time.Now().Unix() - ms.TimeSeriesInterval; err == nil && blk.GetTimestamp() > t; {
		bs := &metricspb.BlockStats{NumTransactions: uint64(len(blk.GetTransactions())), Height: blk.GetHeight()}
		if !cons.Produced(blk) {
			bs.NumTransactions = 0
		}
		stats = append(stats, bs)
		blk, err = it.Next()
	}
	return util.ReverseSlice(stats).([]*metricspb.BlockStats)
}
