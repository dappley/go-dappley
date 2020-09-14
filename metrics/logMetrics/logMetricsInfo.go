package logMetrics

import (
	"encoding/json"
	"github.com/dappley/go-dappley/common/log"
	"github.com/dappley/go-dappley/logic/blockproducer"
	"os"
	"runtime"
	"time"

	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/transactionpool"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/rpc"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type memStat struct {
	CurrentProcessMemInUse   uint64  `json:"currentProcessMemInUse"`
	CurrentProcessMemPercent float32 `json:"currentProcessMemPercent"`
	TotalProcessMemInUse     uint64  `json:"totalProcessMemInUse"`
	TotalProcessMemPercent   float64 `json:"totalProcessMemPercent"`
	SystemMem                uint64  `json:"systemMem"`
}

type cpuStat struct {
	CurrentProcessCpuPercent float64 `json:"currentProcessCpuPercent"`
	TotalProcessCpuPercent   float64 `json:"totalProcessCpuPercent"`
	TotalCoreNum             int     `json:"totalCoreNum"`
}

type diskStat struct {
	Used        uint64  `json:"used"`
	UsedChange  uint64  `json:"UsedChange"`
	UsedPercent float64 `json:"usedPercent"`
	ReadBytes   uint64  `json:"readBytes"`
	WriteBytes  uint64  `json:"writeBytes"`
}

func getMemoryStats() interface{} {
	vm, err := mem.VirtualMemory()
	if err != nil {
		logger.Warn(err)
		return nil
	}

	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		logger.Warn(err)
		return nil
	}

	memInfo, err := proc.MemoryInfo()
	if err != nil {
		logger.Warn(err)
		return nil
	}

	memPercent, err := proc.MemoryPercent()
	if err != nil {
		logger.Warn(err)
		return nil
	}

	stats := &runtime.MemStats{}
	runtime.ReadMemStats(stats)
	return memStat{memInfo.RSS, memPercent, vm.Used, vm.UsedPercent, vm.Total}
}

func getCPUPercent() interface{} {
	cpuInfo, err := cpu.Percent(time.Second, true)
	if err != nil {
		logger.Warn(err)
		return nil
	}
	coreNum := len(cpuInfo)
	cpuTotalPercent := 0.0
	for _, v := range cpuInfo {
		cpuTotalPercent += v
	}

	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		logger.Warn(err)
		return nil
	}

	percentageUsed, err := proc.CPUPercent()
	if err != nil {
		logger.Warn(err)
		return nil
	}

	return cpuStat{percentageUsed, cpuTotalPercent, coreNum}
}

func getDiskStat(diskInfoMap map[string]int) interface{} {
	diskUsage, err := disk.Usage("/")
	if err != nil {
		logger.Warn(err)
		return nil
	}
	disk, err := disk.IOCounters()
	if err != nil {
		logger.Warn(err)
		return nil
	}
	diskUsageChange := diskUsage.Used - uint64(diskInfoMap["preUsed"])
	diskInfoMap["preUsed"] = int(diskUsage.Used)
	return diskStat{diskUsage.Used, diskUsageChange,diskUsage.UsedPercent, disk["disk0"].WriteBytes, disk["disk0"].ReadBytes}
}

func getTransactionPoolSize() interface{} {
	return transactionpool.MetricsTransactionPoolSize.Count()
}

type RequestStats struct {
	Concurrent int64   `json:"concurrent"`
	CostTime   float64 `json:"costTime"`
	Qps        float64 `json:"qps"`
}

func getTxRequestStats() interface{} {
	reqStatsMap := make(map[string]RequestStats)
	for k, v := range rpc.RpcReqMetricsMap {
		reqStatsMap[k] = RequestStats{Concurrent: v.GetConcurrentNum(), CostTime: v.GetResponseTime(), Qps: v.GetRequestPerSecond()}
	}
	return reqStatsMap
}

type TxFromMinerRequestStats struct {
	Concurrent int64   `json:"concurrent"`
	CostTime   float64 `json:"costTime"`
	Qps        float64 `json:"qps"`
}

type ForkInfo struct {
	NumForks    int64 `json:"numForks"`
	LongestFork int64 `json:"longestFork"`
}

type BlockStat struct {
	TxPoolSize       int64   `json:"txPoolSize"`
	Height           uint64  `json:"height"`
	TxAddToBlockCost float64 `json:"txAddToBlockCost"`
}

func getBlockStats(bc *lblockchain.Blockchain) interface{} {
	bs := BlockStat{Height: bc.GetMaxHeight(), TxPoolSize: getTransactionPoolSize().(int64), TxAddToBlockCost: blockproducer.TxAddToBlockCost.Snapshot().Mean()}

	return bs
}

type Network struct {
	//BroadCastTime        float64 `json:"broadcastTime"`
	ConnectionTypeInNum  int64  `json:"connectionTypeInNum"`
	ConnectionTypeOutNum int64  `json:"connectionTypeOutNum"`
	BytesSent            uint64 `json:"bytesSent"`   // number of bytes sent
	BytesRecv            uint64 `json:"bytesRecv"`   // number of bytes received
	PacketsSent          uint64 `json:"packetsSent"` // number of packets sent
	PacketsRecv          uint64 `json:"packetsRecv"` // number of packets received
}

func getNetWorkStats() interface{} {
	net, err := net.IOCounters(false)
	if err != nil {
		logger.Warn(err)
		return nil
	}
	return Network{network.ConnectionTypeInNum.Snapshot().Value(),
		network.ConnectionTypeOutNum.Snapshot().Value(),
		net[0].BytesSent,
		net[0].BytesRecv,
		net[0].PacketsSent,
		net[0].PacketsRecv}
}

type MetricsInfo struct {
	Metrics map[string]interface{} `json:"metrics"`
}

func (mi *MetricsInfo) Add(name string, value interface{}) {
	mi.Metrics[name] = value
}

func (mi *MetricsInfo) ToJsonString() string {
	bt, _ := json.Marshal(mi.Metrics)
	return string(bt)
}

func NewMetricsInfo() *MetricsInfo {
	mi := &MetricsInfo{Metrics: make(map[string]interface{})}
	return mi
}

var mi = NewMetricsInfo()

func LogMetricsInfo(bc *lblockchain.Blockchain) {
	diskInfoMap := make(map[string]int)
	blkHeight := bc.GetMaxHeight()
	interval := viper.GetInt64("metrics.interval")

	go func() {
		defer log.CrashHandler()

		tick := time.NewTicker(time.Duration(interval) * time.Millisecond)
		for {
			select {
			case <-tick.C:
				if bc.GetMaxHeight() > blkHeight{
					blkHeight = bc.GetMaxHeight()
					mi.Metrics["cpu"] = getCPUPercent()
					mi.Metrics["memory"] = getMemoryStats()
					mi.Metrics["disk"] = getDiskStat(diskInfoMap)
					mi.Metrics["block"] = getBlockStats(bc)
					mi.Metrics["txRequest"] = getTxRequestStats()
					mi.Metrics["network"] = getNetWorkStats()
					logger.WithField("metrics", mi.ToJsonString()).Infof("")
					rpc.MetricsInfo = mi.ToJsonString()
				}
			}
		}
	}()
	logger.Debugf("start to log metrics info, interval %v", interval)
}