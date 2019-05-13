package metrics

import (
	"fmt"
	"net"
	"net/http"

	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	logger "github.com/sirupsen/logrus"
)

func startServer(listener net.Listener) {
	err := http.Serve(listener, http.DefaultServeMux)
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
