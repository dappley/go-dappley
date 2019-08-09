package rpc

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/logic/transaction_pool"

	"github.com/shirou/gopsutil/process"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/metrics"
	metricspb "github.com/dappley/go-dappley/metrics/pb"
	"github.com/dappley/go-dappley/network"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/util"
)

type MetricsService struct {
	node *network.Node
	bm   *blockchain_logic.BlockchainManager
	ds   *metrics.DataStore
	*MetricsServiceConfig
	RPCPort uint32
}

func NewMetricsService(node *network.Node, bm *blockchain_logic.BlockchainManager, config *MetricsServiceConfig, RPCPort uint32) *MetricsService {
	return (&MetricsService{node: node, bm: bm, MetricsServiceConfig: config, RPCPort: RPCPort}).init()
}

func (ms *MetricsService) init() *MetricsService {
	if err := ms.node.GetNetwork().StartNewPingService(time.Duration(ms.PollingInterval) * time.Second); err != nil {
		logger.WithError(err).Error("MetricsService: Unable to start new ping service.")
	}
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

func (ms *MetricsService) RpcGetNodeConfig(ctx context.Context, request *rpcpb.MetricsServiceRequest) (*rpcpb.GetNodeConfigResponse, error) {
	return ms.getNodeConfig(), nil
}

func (ms *MetricsService) RpcSetNodeConfig(ctx context.Context, request *rpcpb.SetNodeConfigRequest) (*rpcpb.GetNodeConfigResponse, error) {
	for _, v := range request.GetUpdatedConfigs() {
		if !util.InProtoEnum("rpcpb.SetNodeConfigRequest_ConfigType", v.String()) {
			return nil, status.Error(codes.InvalidArgument, "unrecognized node configuration type")
		}

		if v == rpcpb.SetNodeConfigRequest_MAX_PRODUCERS || v == rpcpb.SetNodeConfigRequest_PRODUCERS {
			cons, ok := ms.bm.Getblockchain().GetConsensus().(*consensus.DPOS)
			if !ok {
				return nil, status.Error(codes.InvalidArgument, "producer properties are only supported for DPOS Consensus")
			}

			if v == rpcpb.SetNodeConfigRequest_PRODUCERS {
				maxProducers := cons.GetDynasty().GetMaxProducers()
				for _, v := range request.GetUpdatedConfigs() {
					if v == rpcpb.SetNodeConfigRequest_MAX_PRODUCERS {
						maxProducers = int(request.GetMaxProducers())
					}
				}
				if err := cons.GetDynasty().CanSetProducers(request.GetProducers(), maxProducers); err != nil {
					return nil, status.Error(codes.InvalidArgument, err.Error())
				}
			}
		}
	}

	for _, v := range request.GetUpdatedConfigs() {
		switch v {
		case rpcpb.SetNodeConfigRequest_TX_POOL_LIMIT:
			ms.bm.Getblockchain().GetTxPool().SetSizeLimit(request.GetTxPoolLimit())
		case rpcpb.SetNodeConfigRequest_BLK_SIZE_LIMIT:
			ms.bm.Getblockchain().SetBlockSizeLimit(int(request.GetBlkSizeLimit()))
		case rpcpb.SetNodeConfigRequest_MAX_CONN_OUT:
			ms.node.GetNetwork().GetStreamManager().GetConnectionManager().SetMaxConnectionOutCount(int(request.GetMaxConnectionOut()))
		case rpcpb.SetNodeConfigRequest_MAX_CONN_IN:
			ms.node.GetNetwork().GetStreamManager().GetConnectionManager().SetMaxConnectionInCount(int(request.GetMaxConnectionIn()))
		case rpcpb.SetNodeConfigRequest_MAX_PRODUCERS:
			ms.bm.Getblockchain().GetConsensus().(*consensus.DPOS).
				GetDynasty().SetMaxProducers(int(request.GetMaxProducers()))
		case rpcpb.SetNodeConfigRequest_PRODUCERS:
			ms.bm.Getblockchain().GetConsensus().(*consensus.DPOS).
				GetDynasty().SetProducers(request.GetProducers())
		}
	}
	return ms.getNodeConfig(), nil
}

func (ms *MetricsService) getNodeConfig() *rpcpb.GetNodeConfigResponse {
	return &rpcpb.GetNodeConfigResponse{
		TxPoolLimit:      ms.bm.Getblockchain().GetTxPool().GetSizeLimit(),
		BlkSizeLimit:     uint32(ms.bm.Getblockchain().GetBlockSizeLimit()),
		MaxConnectionOut: uint32(ms.node.GetNetwork().GetStreamManager().GetConnectionManager().GetMaxConnectionOutCount()),
		MaxConnectionIn:  uint32(ms.node.GetNetwork().GetStreamManager().GetConnectionManager().GetMaxConnectionInCount()),
		ProducerAddress:  ms.bm.Getblockchain().GetConsensus().GetProducerAddress(),
		Producers:        ms.bm.Getblockchain().GetConsensus().GetProducers(),
		MaxProducers:     ms.getMaxProducers(),
		IpfsAddresses:    ms.node.GetIPFSAddresses(),
		RpcPort:          ms.RPCPort,
	}
}

func (ms *MetricsService) getMaxProducers() uint32 {
	if dpos, ok := ms.bm.Getblockchain().GetConsensus().(*consensus.DPOS); ok {
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
		TransactionPoolSize: transaction_pool.MetricsTransactionPoolSize.Count(),
	}
}

func (ms *MetricsService) getNumForksInBlockChain() metricspb.StatValue {
	numForks, longestFork := ms.bm.NumForks()
	return &metricspb.Stat_ForkStats{
		ForkStats: &metricspb.ForkStats{
			NumForks:    numForks,
			LongestFork: longestFork,
		},
	}
}

func (ms *MetricsService) getBlockStats() []*metricspb.BlockStats {
	stats := make([]*metricspb.BlockStats, 0)
	it := ms.bm.Getblockchain().Iterator()
	cons := ms.bm.Getblockchain().GetConsensus()
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
