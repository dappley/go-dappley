package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/blockproducerinfo"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/transactionpool"

	"github.com/dappley/go-dappley/config"
	configpb "github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/logic/downloadmanager"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	logger "github.com/sirupsen/logrus"
)

const (
	nodeDbPath          = "db/temp.db"
	reportFilePath      = "report.csv"
	genesisAddrTest     = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	genesisFilePathTest = "../conf/genesis.conf"
	confDir             = "../../../storage/fakeFileLoaders"
	testport1           = 10851
	testport2           = 10852
	cdBtwTest           = time.Second * 10
)

func main() {

	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	logger.SetLevel(logger.InfoLevel)

	logger.Warn("Please make sure you have moved the test database files to benchmark_tool/db folder!")
	logger.Info("Type enter to continue...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	files, err := ioutil.ReadDir("./db")
	if err != nil {
		logger.WithError(err).Panic("can not read db files")
	}
	if len(files) == 0 {
		logger.Warn("No db files are found. Exiting...")
		return
	}

	var elapsed time.Duration
	var blkHeight uint64
	var numOfTx int

	filePath := "report_" + time.Now().String() + ".csv"

	initRecordFile(filePath)

	for _, file := range files {
		elapsed, blkHeight, numOfTx = runTest("./db/" + file.Name())
		recordResult(filePath, file.Name(), elapsed, blkHeight, numOfTx)
		logger.WithFields(logger.Fields{
			"time_elapsed": elapsed,
			"ave_blk_time": elapsed / time.Duration(blkHeight),
			"ave_tx_time":  elapsed / time.Duration(blkHeight) / time.Duration(numOfTx),
		}).Info("Test Finished")
		time.Sleep(cdBtwTest)
	}
}

func initRecordFile(filePath string) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Panic("Open file failed while recording failed transactions")
	}
	w := csv.NewWriter(f)
	vmStat, err := mem.VirtualMemory()
	cpuStat, err := cpu.Info()
	w.Write([]string{
		"", "", "", "", "", "", "Memory",
		vmStat.String(),
	})
	w.Write([]string{
		"", "", "", "", "", "", "Cpu",
		cpuStat[0].String(),
	})
	w.Write([]string{
		"Timestamp",
		"Db file",
		"Number of blocks",
		"Number of transactions per block",
		"Total time elapsed",
		"Average block time",
		"Average transaction time",
	})
	w.Flush()
}

func recordResult(filePath string, dbfileName string, elapsed time.Duration, blkHeight uint64, numOfTx int) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Panic("Open file failed while recording failed transactions")
	}
	w := csv.NewWriter(f)
	w.Write([]string{
		time.Now().String(),
		dbfileName,
		fmt.Sprint(blkHeight),
		fmt.Sprint(numOfTx),
		elapsed.String(),
		(elapsed / time.Duration(blkHeight)).String(),
		(elapsed / time.Duration(blkHeight) / time.Duration(numOfTx)).String(),
	})
	w.Flush()
}

func runTest(fileName string) (time.Duration, uint64, int) {
	logger.WithFields(logger.Fields{
		"db_file_path": fileName,
	}).Info("Test starts...")
	defer os.RemoveAll(nodeDbPath)
	db1 := storage.OpenDatabase(fileName)
	defer db1.Close()
	db2 := storage.OpenDatabase(nodeDbPath)
	defer db2.Close()
	rfl1 := storage.NewRamFileLoader(confDir, "test1.conf")
	defer rfl1.DeleteFolder()
	rfl2 := storage.NewRamFileLoader(confDir, "test2.conf")
	defer rfl2.DeleteFolder()

	bm1, node1 := prepareNode(db1, rfl1.File)
	bm2, node2 := prepareNode(db2, rfl2.File)
	downloadmanager.NewDownloadManager(node1, bm1, 0, nil)
	downloadmanager.NewDownloadManager(node2, bm2, 0, nil)

	node1.Start(testport1, "")
	defer node1.Stop()
	node2.Start(testport2, "")
	defer node2.Stop()

	node1.GetNetwork().ConnectToSeed(node2.GetHostPeerInfo())

	blkHeight := bm1.Getblockchain().GetMaxHeight()
	tailBlock, _ := bm1.Getblockchain().GetTailBlock()
	numOfTx := len(tailBlock.GetTransactions())

	logger.WithFields(logger.Fields{
		"blk_height": blkHeight,
		"num_of_tx":  numOfTx,
	}).Info("Start Downloading...")

	time.Sleep(time.Second)

	start := time.Now()
	bm2.RequestDownloadBlockchain()
	elapsed := time.Since(start)

	logger.WithFields(logger.Fields{
		"time_elapsed": elapsed,
		"ave_blk_time": elapsed / time.Duration(blkHeight),
		"ave_tx_time":  elapsed / time.Duration(blkHeight) / time.Duration(numOfTx),
	}).Info("Downloading ends... Cleaning up files...")

	return elapsed, blkHeight, numOfTx
}

func prepareNode(db storage.Storage, rfl *storage.FileLoader) (*lblockchain.BlockchainManager, *network.Node) {
	genesisConf := &configpb.DynastyConfig{}
	config.LoadConfig(genesisFilePathTest, genesisConf)
	maxProducers := (int)(genesisConf.GetMaxProducers())
	dynasty := consensus.NewDynastyWithConfigProducers(genesisConf.GetProducers(), maxProducers)
	conss := consensus.NewDPOS(blockproducerinfo.NewBlockProducerInfo(""))
	conss.SetDynasty(dynasty)
	node := network.NewNode(rfl, nil)
	txPoolLimit := uint32(2000)
	txPool := transactionpool.NewTransactionPool(node, txPoolLimit)
	bc, err := lblockchain.GetBlockchain(db, conss, txPool, 1000000)
	if err != nil {
		bc, err = logic.CreateBlockchain(account.NewAddress(genesisAddrTest), db, conss, txPool, 1000000)
		if err != nil {
			logger.Panic(err)
		}
	}
	bc.SetState(blockchain.BlockchainInit)
	LIBBlk, _ := bc.GetLIB()
	bm := lblockchain.NewBlockchainManager(bc, blockchain.NewBlockPool(LIBBlk), node, conss)
	downloadmanager.NewDownloadManager(node, bm, len(conss.GetProducers()), nil)

	return bm, node
}
