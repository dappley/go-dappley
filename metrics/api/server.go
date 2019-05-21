package metrics

import (
	"expvar"
	"fmt"
	"net"
	"net/http"

	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/rs/cors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	logger "github.com/sirupsen/logrus"
)

func init() {
	expvar.Publish("memory.stats", expvar.Func(func () interface{} {
		res, err := mem.VirtualMemory()
		if err != nil {
			logger.Warn(err)
			return nil
		}

		return res
	}))

	expvar.Publish("cpu.stats", expvar.Func(func () interface{} {
		res, err := cpu.Times(true)
		if err != nil {
			logger.Warn(err)
			return nil
		}

		return res
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

	go startServer(listener)

	return listener.Addr().(*net.TCPAddr).Port
}
