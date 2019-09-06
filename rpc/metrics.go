package rpc

import (
	"github.com/dappley/go-dappley/metrics"
	gometrics "github.com/rcrowley/go-metrics"
)

var(
	txRequestStats = NewTxRequestStats("txRequestSend")
	txRequestFromMinerStats = NewTxRequestStats("txRequestSendFromMiner")
	RpcReqMetricsMap = map[string]ReqStatsFunc{
		txRequestStats.name:txRequestStats,
		txRequestFromMinerStats.name:txRequestFromMinerStats,
	}
)

type ReqStats struct {
	concurrentCounter gometrics.Counter
	responseTime gometrics.Histogram
	requestPerSec gometrics.Meter
}

type ReqStatsFunc interface {
	GetConcurrentNum() int64
	GetResponseTime() float64
	GetRequestPerSecond() float64
}


type TxRequestStats struct{
		ReqStats
		name string
}

func NewTxRequestStats(name string) *TxRequestStats{
	trs := new(TxRequestStats)
	trs.concurrentCounter = metrics.NewCounter(name + ".concurrent.number")
	trs.responseTime = metrics.NewHistogram(name + ".time.cost")
	trs.requestPerSec = metrics.NewMeter(name + ".qps")
	trs.name = name
	return trs
}


func (trs *TxRequestStats)GetConcurrentNum() int64{
	return trs.concurrentCounter.Snapshot().Count()
}

func (trs *TxRequestStats)GetResponseTime() float64{
	return trs.responseTime.Snapshot().Mean()
}

func (trs *TxRequestStats)GetRequestPerSecond() float64{
	return trs.requestPerSec.Snapshot().Rate1()
}


