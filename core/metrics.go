package core

import (
	"github.com/dappley/go-dappley/metrics"
)

// Metrics for core
var (
	MetricsTransactionPoolSize = metrics.NewCounter("dap.txPool.currSize")
)
