package main

import (
	"bufio"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

const(
	nodeDbPath = "db/temp.db"
	genesisAddrTest = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	genesisFilePathTest = "../conf/genesis.conf"
	testport1 = 10851
	testport2 = 10852
)

func main() {

	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	logger.SetLevel(logger.InfoLevel)

	logger.Warn("Please make sure you have moved the test database file to benchmark/db folder!")
	logger.Warn("Please make sure you have renamed your database file to test.db!")
	logger.Info("Please enter the database file name")
	reader := bufio.NewReader(os.Stdin)
	fileName, _ := reader.ReadString('\n')
	fileName = strings.TrimSuffix(fileName, "\n")
	fileName = "db/"+fileName

	var totalElaspedTime time.Duration
	var elapsed time.Duration
	var blkHeight uint64
	var numOfTx int

	repeat := 1

	for i:=0;i<repeat;i++ {
		elapsed, blkHeight, numOfTx = runTest(fileName)
		totalElaspedTime+=elapsed
		time.Sleep(time.Second)
	}
	elapsed = totalElaspedTime/time.Duration(repeat)
	logger.WithFields(logger.Fields{
		"time_elapsed" : elapsed,
		"ave_blk_time"  : elapsed/time.Duration(blkHeight),
		"ave_tx_time"  : elapsed/time.Duration(blkHeight)/time.Duration(numOfTx),
	}).Info("Test Finished")

}

func runTest(fileName string) (time.Duration, uint64, int){
	db1 := storage.OpenDatabase(fileName)
	defer db1.Close()
	db2 := storage.OpenDatabase(nodeDbPath)
	defer db2.Close()

	bc, node1 := prepareNode(db1)
	bc2, node2 := prepareNode(db2)

	node1.Start(testport1)
	defer node1.Stop()
	node2.Start(testport2)
	defer node2.Stop()

	node1.GetPeerManager().AddAndConnectPeer(node2.GetInfo())

	blkHeight := bc.GetMaxHeight()
	tailBlock,_ := bc.GetTailBlock()
	numOfTx := len(tailBlock.GetTransactions())

	logger.WithFields(logger.Fields{
		"blk_height" : blkHeight,
		"num_of_tx"  : numOfTx,
	}).Info("Start Downloading...")

	time.Sleep(time.Second)

	start := time.Now()
	downloadBlocks(node2, bc2)
	elapsed := time.Since(start)

	logger.WithFields(logger.Fields{
		"time_elapsed" : elapsed,
		"ave_blk_time"  : elapsed/time.Duration(blkHeight),
		"ave_tx_time"  : elapsed/time.Duration(blkHeight)/time.Duration(numOfTx),
	}).Info("Downloading ends... Cleaning up files...")

	os.RemoveAll(nodeDbPath)

	return elapsed, blkHeight, numOfTx
}



func prepareNode(db storage.Storage) (*core.Blockchain, *network.Node){
	genesisConf := &configpb.DynastyConfig{}
	config.LoadConfig(genesisFilePathTest, genesisConf)
	maxProducers := (int)(genesisConf.GetMaxProducers())
	dynasty := consensus.NewDynastyWithConfigProducers(genesisConf.GetProducers(), maxProducers)
	conss := consensus.NewDPOS()
	conss.SetDynasty(dynasty)
	txPoolLimit:=uint32(2000)
	bc, err := core.GetBlockchain(db, conss, txPoolLimit, nil)
	if err != nil {
		bc, err = logic.CreateBlockchain(core.NewAddress(genesisAddrTest), db, conss, txPoolLimit, nil)
		if err != nil {
			logger.Panic(err)
		}
	}

	bc.SetState(core.BlockchainInit)
	node := network.NewNode(bc, core.NewBlockPool(0))
	return bc, node
}

func downloadBlocks(node *network.Node, bc *core.Blockchain) {
	downloadManager := node.GetDownloadManager()
	finishChan := make(chan bool, 1)
	bc.SetState(core.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishChan)
	<-finishChan
	bc.SetState(core.BlockchainReady)
}