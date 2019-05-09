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
		logger.Panic("Unable to start metrics api server: ", err)
	}
}

// starts the metrics api
func StartAPI(port uint32) int {

	// expose metrics at /debug/metrics
	exp.Exp(metrics.DefaultRegistry)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		logger.Panic(err)
	}

	logger.Info(fmt.Sprintf("Start metrics api at %v/debug/metrics ...", listener.Addr()))

	go startServer(listener)

	return listener.Addr().(*net.TCPAddr).Port
}
