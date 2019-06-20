package core

import (
	dapmetrics "github.com/dappley/go-dappley/metrics"
)

// Metrics for core
var (
	MetricsTransactionPoolSize = dapmetrics.NewCounter("dap.txPool.currSize")
)
