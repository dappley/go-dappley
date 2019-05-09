package metrics

import (
	"fmt"
	"net/http"

	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	logger "github.com/sirupsen/logrus"
)

// starts the metrics api
func StartAPI(port uint32) {

	// expose metrics at /debug/metrics
	exp.Exp(metrics.DefaultRegistry)

	logger.Info("Starting metrics api...")
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), http.DefaultServeMux)

	if err != nil {
		logger.Panic("Unable to start metrics api server: ", err)
	}
}
