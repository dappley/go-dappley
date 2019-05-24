package metrics

import (
	"expvar"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/rs/cors"
	"github.com/shirou/gopsutil/process"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
)



func initAPI(interval, pollingInterval int64) {

	pid := int32(os.Getpid())

	getCPUPercent := expvar.Func(func() interface{} {
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
	})

	expvar.Publish("dapp.cpu.percent", getCPUPercent)

	ds := newDataStore(int(interval / pollingInterval), time.Duration(pollingInterval) * time.Second)

	_ = ds.registerNewMetric("dapp.cpu.percent", getCPUPercent)

	_ = ds.registerNewMetric("dapp.txpool.size", func() interface{} {
		return core.MetricsTransactionPoolSize.Count()
	})

	type MemStat struct {
		HeapInuse uint64 `json:"heapInUse"`
		HeapSys uint64 `json:"heapSys"`
	}

	_ = ds.registerNewMetric("dapp.memstats", func() interface{} {
		stats := &runtime.MemStats{}
		runtime.ReadMemStats(stats)
		return MemStat{stats.HeapInuse, stats.HeapSys}
	})

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
func StartAPI(conf *configpb.NodeConfig) int {
	// expose metrics at /debug/metrics
	exp.Exp(metrics.DefaultRegistry)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", conf.GetMetricsHost(), conf.GetMetricsPort()))

	if err != nil {
		logger.Panic(err)
	}

	logger.WithFields(logger.Fields{
		"endpoint": fmt.Sprintf("%v/debug/metrics", listener.Addr()),
	}).Info("Metrics: API starts...")

	initAPI(conf.GetMetricsInterval(), conf.GetMetricsPollingInterval())

	go startServer(listener)

	return listener.Addr().(*net.TCPAddr).Port
}
