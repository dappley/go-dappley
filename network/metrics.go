package network

import "github.com/dappley/go-dappley/metrics"

var(
	ConnectionTypeInNum = metrics.NewGauge("connection.in.num")
	ConnectionTypeOutNum = metrics.NewGauge("connection.out.num")
	BroadCastCostTime = metrics.NewHistogram("broadcast.cost.time")
)