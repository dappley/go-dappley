package core

import "github.com/dappley/go-dappley/metrics"

// Metrics for core
var (
	// tx metrics
	MetricsTxDoubleSpend             = dapmetrics.NewCounter("dap.txpool.doublespend")
	MetricsInvalidTx				 = dapmetrics.NewCounter("dap.txPool.invalidtx")

)
