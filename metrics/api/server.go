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

	"github.com/dappley/go-dappley/core"
)

func initAPI() {

	pid := int32(os.Getpid())

	expvar.Publish("dapp.cpu.percent", expvar.Func(func() interface{} {
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
	}))

	// 2 hr of data at 5 sec interval
	ds := NewDataStore(1440, 5 * time.Second)

	ds.RegisterNewMetric("dapp.cpu.percent", func() interface{} {
		proc, err := process.NewProcess(pid)
		if err != nil {
			logger.Warn(err)
		}

		percentageUsed, err := proc.CPUPercent()
		if err != nil {
			logger.Warn(err)
		}

		return percentageUsed
	})

	ds.RegisterNewMetric("dapp.txpool.size", func() interface{} {
		return core.MetricsTransactionPoolSize.Count()
	})

	type MemStat struct {
		HeapInuse uint64 `json:"heapInUse"`
		HeapSys uint64 `json:"heapSys"`
	}

	ds.RegisterNewMetric("dapp.memstats", func() interface{} {
		stats := &runtime.MemStats{}
		runtime.ReadMemStats(stats)
		return MemStat{stats.HeapInuse, stats.HeapSys}
	})

	ds.StartUpdate()

	expvar.Publish("stats", expvar.Func(func() interface{} {
		return ds
	}))
}

func startServer(listener net.Listener) {
	handler := cors.New(cors.Options{AllowedOrigins: []string{"*"}})
	err := http.Serve(listener, handler.Handler(http.DefaultServeMux))
	if err != nil {
		logger.WithError(err).Panic("Metrics: unable to start api server.")
	}
}

// starts the metrics api
func StartAPI(host string, port uint32) int {
	// expose metrics at /debug/metrics
	exp.Exp(metrics.DefaultRegistry)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))

	if err != nil {
		logger.Panic(err)
	}

	logger.WithFields(logger.Fields{
		"endpoint": fmt.Sprintf("%v/debug/metrics", listener.Addr()),
	}).Info("Metrics: API starts...")

	initAPI()

	go startServer(listener)

	return listener.Addr().(*net.TCPAddr).Port
}
