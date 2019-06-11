package core

import (
	dapmetrics "github.com/dappley/go-dappley/metrics"
)

// Metrics for core
var (
	MetricsBlockStats          = dapmetrics.NewBlockStatsTracker(50)
	MetricsTransactionPoolSize = dapmetrics.NewCounter("dap.txPool.currSize")
)
