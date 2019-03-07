package core

import "github.com/dappley/go-dappley/metrics"

// Metrics for core
var (
	// tx metrics
	MetricsInvalidTx				 = dapmetrics.NewCounter("dap.txPool.invalidtx")
)
