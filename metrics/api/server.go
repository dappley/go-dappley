package metrics

import (
	"expvar"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/rs/cors"
	"github.com/shirou/gopsutil/process"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
)

type memStat struct {
	HeapInuse uint64 `json:"heapInUse"`
	HeapSys   uint64 `json:"heapSys"`
}

func getMemoryStats() interface{} {
	stats := &runtime.MemStats{}
	runtime.ReadMemStats(stats)
	return memStat{stats.HeapInuse, stats.HeapSys}
}

func getCPUPercent() interface{} {
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

	return percentageUsed
}

func getTransactionPoolSize() interface{} {
	return core.MetricsTransactionPoolSize.Count()
}

type peerInfo struct {
	Info    *peerstore.PeerInfo `json:"info"`
	Latency string              `json:"latency"`
}

func getConnectedPeersFunc(node *network.Node) expvar.Func {
	getConnectedPeers := func() interface{} {
		var peers []peerInfo
		for _, peer := range node.GetPeerManager().CloneStreamsToPeerInfoSlice() {
			peers = append(peers, peerInfo{&peerstore.PeerInfo{peer.PeerId, peer.Addrs}, peer.Latency.String()})
		}
		return peers
	}
	return getConnectedPeers
}

func initPeerMetrics(node *network.Node, interval int64) {
	if node != nil {
		expvar.Publish("peers", getConnectedPeersFunc(node))
		node.GetPeerManager().StartNewPingService(time.Duration(interval) * time.Second)
	}
}

func initIntervalMetrics(interval, pollingInterval int64) {
	ds := newDataStore(int(interval/pollingInterval), time.Duration(pollingInterval)*time.Second)

	_ = ds.registerNewMetric("dapp.cpu.percent", getCPUPercent)
	_ = ds.registerNewMetric("dapp.txpool.size", getTransactionPoolSize)
	_ = ds.registerNewMetric("dapp.memstats", getMemoryStats)

	expvar.Publish("dapp.cpu.percent", expvar.Func(getCPUPercent))
	expvar.Publish("stats", ds)
	ds.startUpdate()
}

func startServer(listener net.Listener) {
	handler := cors.New(cors.Options{AllowedOrigins: []string{"*"}})
	err := http.Serve(listener, handler.Handler(http.DefaultServeMux))
	if err != nil {
		logger.WithError(err).Panic("Metrics: unable to start api server.")
	}
}

// starts the metrics api
func StartAPI(node *network.Node, host string, port uint32, interval int64, pollingInterval int64) int {
	// expose metrics at /debug/metrics
	exp.Exp(metrics.DefaultRegistry)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))

	if err != nil {
		logger.Panic(err)
	}

	logger.WithFields(logger.Fields{
		"endpoint": fmt.Sprintf("%v/debug/metrics", listener.Addr()),
	}).Info("Metrics: API starts...")

	initPeerMetrics(node, pollingInterval)
	initIntervalMetrics(interval, pollingInterval)

	go startServer(listener)

	return listener.Addr().(*net.TCPAddr).Port
}
