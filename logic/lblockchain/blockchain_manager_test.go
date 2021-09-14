package lblockchain

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/blockproducerinfo"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/lblock"
	blockchainMock "github.com/dappley/go-dappley/logic/lblockchain/mocks"
	lblockchainpb "github.com/dappley/go-dappley/logic/lblockchain/pb"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/logic/transactionpool"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
)

const confDir = "../../storage/fakeFileLoaders/"
const senderNodePort = 10298
const requesterNodePort = 10299

func TestBlockChainManager_NumForks(t *testing.T) {
	// create BlockChain
	bc := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 100), 100)
	blk, err := bc.GetTailBlock()
	require.Nil(t, err)

	b1 := block.NewBlockWithRawInfo(nil, blk.GetHash(), 1, 0, 1, nil)
	b3 := block.NewBlockWithRawInfo(nil, b1.GetHash(), 3, 0, 2, nil)
	b3.SetHash(lblock.CalculateHash(b3))
	b6 := block.NewBlockWithRawInfo(nil, b3.GetHash(), 6, 0, 3, nil)
	b6.SetHash(lblock.CalculateHash(b6))

	err = bc.AddBlockContextToTail(&BlockContext{Block: b1, UtxoIndex: lutxo.NewUTXOIndex(nil), State: scState.NewScState(bc.GetUtxoCache())})
	require.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b3, UtxoIndex: lutxo.NewUTXOIndex(nil), State: scState.NewScState(bc.GetUtxoCache())})
	require.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b6, UtxoIndex: lutxo.NewUTXOIndex(nil), State: scState.NewScState(bc.GetUtxoCache())})
	require.Nil(t, err)

	// create first fork of height 3
	b2 := block.NewBlockWithRawInfo(nil, b1.GetHash(), 2, 0, 2, nil)
	b2.SetHash(lblock.CalculateHash(b2))

	b4 := block.NewBlockWithRawInfo(nil, b2.GetHash(), 4, 0, 3, nil)
	b4.SetHash(lblock.CalculateHash(b4))

	b5 := block.NewBlockWithRawInfo(nil, b2.GetHash(), 5, 0, 3, nil)
	b5.SetHash(lblock.CalculateHash(b5))

	b7 := block.NewBlockWithRawInfo(nil, b4.GetHash(), 7, 0, 4, nil)
	b7.SetHash(lblock.CalculateHash(b7))

	/*
		              b1
		            b2  b3
		          b4 b5  b6
		        b7
			BlockChain:  Genesis - b1 - b3 - b6
	*/

	bp := blockchain.NewBlockPool(nil)
	bcm := NewBlockchainManager(bc, bp, nil, nil)

	bp.AddBlock(b2)
	require.Equal(t, 1, testGetNumForkHeads(bp))
	bp.AddBlock(b4)
	require.Equal(t, 1, testGetNumForkHeads(bp))
	bp.AddBlock(b5)
	require.Equal(t, 1, testGetNumForkHeads(bp))
	bp.AddBlock(b7)
	require.Equal(t, 1, testGetNumForkHeads(bp))

	// adding block that is not connected to BlockChain should be ignored
	b8 := block.NewBlockWithRawInfo(nil, []byte{9}, 8, 0, 4, nil)
	b8.SetHash(lblock.CalculateHash(b8))
	bp.AddBlock(b8)
	require.Equal(t, 2, testGetNumForkHeads(bp))

	numForks, longestFork := bcm.NumForks()
	require.EqualValues(t, 2, numForks)
	require.EqualValues(t, 3, longestFork)

	// create a new fork off b6
	b9 := block.NewBlockWithRawInfo(nil, b6.GetHash(), 9, 0, 4, nil)
	b9.SetHash(lblock.CalculateHash(b9))

	bp.AddBlock(b9)
	require.Equal(t, 3, testGetNumForkHeads(bp))

	require.ElementsMatch(t,
		[]string{b2.GetHash().String(), b8.GetHash().String(), b9.GetHash().String()}, testGetForkHeadHashes(bp))

	numForks, longestFork = bcm.NumForks()
	require.EqualValues(t, 3, numForks)
	require.EqualValues(t, 3, longestFork)
}

func testGetNumForkHeads(bp *blockchain.BlockPool) int {
	return len(testGetForkHeadHashes(bp))
}

func testGetForkHeadHashes(bp *blockchain.BlockPool) []string {
	var hashes []string
	bp.ForkHeadRange(func(blkHash string, tree *common.TreeNode) {
		hashes = append(hashes, blkHash)
	})
	return hashes
}

func TestGetUTXOIndexAtBlockHash(t *testing.T) {
	genesisAddr := account.NewAddress("##@@")
	genesisBlock := NewGenesisBlock(genesisAddr, transaction.Subsidy)

	// prepareBlockchainWithBlocks returns a blockchain that contains the given blocks with correct utxoIndex in RAM
	prepareBlockchainWithBlocks := func(blks []*block.Block) *Blockchain {
		bc := CreateBlockchain(genesisAddr, storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 128000), 100000)
		for _, blk := range blks {
			err := bc.AddBlockContextToTail(PrepareBlockContext(bc, blk))
			if err != nil {
				logger.Fatal("TestGetUTXOIndexAtBlockHash: cannot add the blocks to blockchain.")
			}
		}
		return bc
	}

	// utxoIndexFromTXs creates a utxoIndex containing all vout of transactions in txs
	utxoIndexFromTXs := func(txs []*transaction.Transaction, cache *utxo.UTXOCache) *lutxo.UTXOIndex {
		utxoIndex := lutxo.NewUTXOIndex(cache)
		utxosMap := make(map[string]*utxo.UTXOTx)
		for _, tx := range txs {
			for i, vout := range tx.Vout {
				utxos, ok := utxosMap[vout.PubKeyHash.String()]
				if !ok {
					newUtxos := utxo.NewUTXOTx()
					utxos = &newUtxos
				}
				utxos.PutUtxo(utxo.NewUTXO(vout, tx.ID, i, utxo.UtxoNormal))
				utxosMap[vout.PubKeyHash.String()] = utxos
			}
		}
		utxoIndex.SetIndexAdd(utxosMap)
		return utxoIndex
	}
	removeUTXOFromBlockchain := func(txs []*transaction.Transaction, cache *utxo.UTXOCache) *lutxo.UTXOIndex {
		utxoIndex := lutxo.NewUTXOIndex(cache)
		utxosMap := make(map[string]*utxo.UTXOTx)
		for _, tx := range txs {
			for i, vout := range tx.Vout {
				utxos, ok := utxosMap[vout.PubKeyHash.String()]
				if !ok {
					newUtxos := utxo.NewUTXOTx()
					utxos = &newUtxos
				}
				utxos.PutUtxo(utxo.NewUTXO(vout, tx.ID, i, utxo.UtxoNormal))
				utxosMap[vout.PubKeyHash.String()] = utxos
			}
		}
		utxoIndex.SetindexRemove(utxosMap)
		return utxoIndex
	}
	acc := account.NewAccount()

	normalTX := ltransaction.NewCoinbaseTX(acc.GetAddress(), "", 1, common.NewAmount(5))
	normalTX2 := transaction.Transaction{
		hash.Hash("normal2"),
		[]transactionbase.TXInput{{normalTX.ID, 0, nil, acc.GetKeyPair().GetPublicKey()}},
		[]transactionbase.TXOutput{{common.NewAmount(5), acc.GetPubKeyHash(), ""}},
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		0,
		transaction.TxTypeNormal,
	}
	abnormalTX := transaction.Transaction{
		hash.Hash("abnormal"),
		[]transactionbase.TXInput{{normalTX.ID, 1, nil, nil}},
		[]transactionbase.TXOutput{{common.NewAmount(5), account.PubKeyHash([]byte("pkh")), ""}},
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		0,
		transaction.TxTypeNormal,
	}
	prevBlock := block.NewBlock([]*transaction.Transaction{}, genesisBlock, "")
	prevBlock.SetHash(lblock.CalculateHash(prevBlock))
	emptyBlock := block.NewBlock([]*transaction.Transaction{}, prevBlock, "")
	emptyBlock.SetHash(lblock.CalculateHash(emptyBlock))
	normalBlock := block.NewBlock([]*transaction.Transaction{&normalTX}, genesisBlock, "")
	normalBlock.SetHash(lblock.CalculateHash(normalBlock))
	normalBlock2 := block.NewBlock([]*transaction.Transaction{&normalTX2}, normalBlock, "")
	normalBlock2.SetHash(lblock.CalculateHash(normalBlock2))
	abnormalBlock := block.NewBlock([]*transaction.Transaction{&abnormalTX}, normalBlock, "")
	abnormalBlock.SetHash(lblock.CalculateHash(abnormalBlock))
	uTXOBlockchain := prepareBlockchainWithBlocks([]*block.Block{normalBlock, normalBlock2})
	err := removeUTXOFromBlockchain([]*transaction.Transaction{&normalTX2}, uTXOBlockchain.GetUtxoCache()).Save()
	if err != nil {
		logger.Fatal("TestGetUTXOIndexAtBlockHash: cannot corrupt the utxoIndex in database.")
	}

	bcs := []*Blockchain{
		prepareBlockchainWithBlocks([]*block.Block{normalBlock}),
		prepareBlockchainWithBlocks([]*block.Block{normalBlock, normalBlock2}),
		CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 128000), 100000),
		prepareBlockchainWithBlocks([]*block.Block{prevBlock, emptyBlock}),
		prepareBlockchainWithBlocks([]*block.Block{normalBlock, normalBlock2}),
		prepareBlockchainWithBlocks([]*block.Block{normalBlock, abnormalBlock}),
		uTXOBlockchain,
	}
	tests := []struct {
		name     string
		bc       *Blockchain
		hash     hash.Hash
		expected *lutxo.UTXOIndex
		err      error
	}{
		{
			name:     "current block",
			bc:       bcs[0],
			hash:     normalBlock.GetHash(),
			expected: utxoIndexFromTXs([]*transaction.Transaction{&normalTX}, bcs[0].GetUtxoCache()),
			err:      nil,
		},
		{
			name:     "previous block",
			bc:       bcs[1],
			hash:     normalBlock.GetHash(),
			expected: utxoIndexFromTXs([]*transaction.Transaction{&normalTX}, bcs[1].GetUtxoCache()), // should not have utxo from normalTX2
			err:      nil,
		},
		{
			name:     "block not found",
			bc:       bcs[2],
			hash:     hash.Hash("not there"),
			expected: lutxo.NewUTXOIndex(bcs[2].GetUtxoCache()),
			err:      ErrBlockDoesNotFound,
		},
		{
			name:     "no txs in blocks",
			bc:       bcs[3],
			hash:     emptyBlock.GetHash(),
			expected: utxoIndexFromTXs(genesisBlock.GetTransactions(), bcs[3].GetUtxoCache()),
			err:      nil,
		},
		{
			name:     "genesis block",
			bc:       bcs[4],
			hash:     genesisBlock.GetHash(),
			expected: utxoIndexFromTXs(genesisBlock.GetTransactions(), bcs[4].GetUtxoCache()),
			err:      nil,
		},
		{
			name:     "utxo not found",
			bc:       bcs[5],
			hash:     normalBlock.GetHash(),
			expected: lutxo.NewUTXOIndex(bcs[5].GetUtxoCache()),
			err:      lutxo.ErrUTXONotFound,
		},
		{
			name:     "corrupted utxoIndex",
			bc:       bcs[6],
			hash:     normalBlock.GetHash(),
			expected: lutxo.NewUTXOIndex(bcs[6].GetUtxoCache()),
			err:      lutxo.ErrUTXONotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := RevertUtxoAndScStateAtBlockHash(tt.bc.GetDb(), tt.bc, tt.hash)
			if !assert.Equal(t, tt.err, err) {
				return
			}
		})
	}
}

func TestBlockchainManager_SetNewDynasty(t *testing.T) {
	// create BlockChain
	bc := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 100), 100)
	_, err := bc.GetTailBlock()
	require.Nil(t, err)

	bp := blockchain.NewBlockPool(nil)

	producerInfo := blockproducerinfo.NewBlockProducerInfo("address")
	conss := consensus.NewDPOS(producerInfo)
	conss.SetDynasty(consensus.NewDynasty([]string{"address1", "address2", "address3"}, 3, 99))
	bcm := NewBlockchainManager(bc, bp, nil, conss)
	assert.Equal(t, []string{"address1", "address2", "address3"}, bcm.consensus.(*consensus.DPOS).GetProducers())

	bcm.SetNewDynasty("address1", "newaddress1", 1, 1) // type 1: change producer
	bcm.SetNewDynasty("address2", "newaddress2", 1, 2) // type 2: append producer
	bcm.SetNewDynasty("address3", "newaddress3", 2, 3) // type 3: delete producer

	bcm.CheckDynast(1)
	expected := []string{"newaddress1", "address2", "address3", "newaddress2"}
	assert.Equal(t, expected, bcm.consensus.(*consensus.DPOS).GetProducers())
	bcm.CheckDynast(2)
	expected = []string{"newaddress1", "address2", "newaddress2"}
	assert.Equal(t, expected, bcm.consensus.(*consensus.DPOS).GetProducers())
}

func TestBlockchainManager_SetNewDynastyByString(t *testing.T) {
	// create BlockChain
	bc := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 100), 100)
	_, err := bc.GetTailBlock()
	require.Nil(t, err)

	bp := blockchain.NewBlockPool(nil)

	producerInfo := blockproducerinfo.NewBlockProducerInfo("address")
	conss := consensus.NewDPOS(producerInfo)
	conss.SetDynasty(consensus.NewDynasty([]string{"address1", "address2", "address3"}, 3, 99))
	bcm := NewBlockchainManager(bc, bp, nil, conss)
	assert.Equal(t, []string{"address1", "address2", "address3"}, bcm.consensus.(*consensus.DPOS).GetProducers())

	json1 := `{"height":1,"addresses":"newaddress1","kind":1}`
	json2 := `{"height":1,"addresses":"newaddress2","kind":2}`
	json3 := `{"height":2,"addresses":"newaddress3","kind":3}`
	bcm.SetNewDynastyByString(json1, "address1") // type 1: change producer
	bcm.SetNewDynastyByString(json2, "address2") // type 2: append producer
	bcm.SetNewDynastyByString(json3, "address3") // type 3: delete producer

	bcm.CheckDynast(1)
	expected := []string{"newaddress1", "address2", "address3", "newaddress2"}
	assert.Equal(t, expected, bcm.consensus.(*consensus.DPOS).GetProducers())
	bcm.CheckDynast(2)
	expected = []string{"newaddress1", "address2", "newaddress2"}
	assert.Equal(t, expected, bcm.consensus.(*consensus.DPOS).GetProducers())
}

func TestBlockchainManager_GetSubscribedTopics(t *testing.T) {
	// create BlockChain
	bc := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 100), 100)
	_, err := bc.GetTailBlock()
	require.Nil(t, err)

	bp := blockchain.NewBlockPool(nil)
	bcm := NewBlockchainManager(bc, bp, nil, nil)

	expected := []string{
		SendBlock,
		RequestBlock,
	}
	assert.Equal(t, expected, bcm.GetSubscribedTopics())
}

func TestBlockchainManager_GetTopicHandler(t *testing.T) {
	// create BlockChain
	bc := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 100), 100)
	_, err := bc.GetTailBlock()
	require.Nil(t, err)

	bp := blockchain.NewBlockPool(nil)
	bcm := NewBlockchainManager(bc, bp, nil, nil)

	// cannot directly compare functions so pointers are used
	sendBlockExpected := reflect.ValueOf(bcm.SendBlockHandler).Pointer()
	sendBlockActual := reflect.ValueOf(bcm.GetTopicHandler(SendBlock)).Pointer()
	assert.Equal(t, sendBlockExpected, sendBlockActual)

	requestBlockExpected := reflect.ValueOf(bcm.RequestBlockHandler).Pointer()
	requestBlockActual := reflect.ValueOf(bcm.GetTopicHandler(RequestBlock)).Pointer()
	assert.Equal(t, requestBlockExpected, requestBlockActual)

	assert.Nil(t, bcm.GetTopicHandler("not a topic"))
}

func TestBlockchainManager_VerifyBlock(t *testing.T) {
	txs := []*transaction.Transaction{
		{
			ID:  []byte{0x74},
			Vin: []transactionbase.TXInput{},
			Vout: []transactionbase.TXOutput{
				{
					Value:      common.NewAmount(10),
					PubKeyHash: account.PubKeyHash{0x5a, 0xf8, 0xbf, 0x23, 0x39, 0x70, 0xf0, 0x9b, 0x65, 0x31, 0x98, 0xca, 0xed, 0x6c, 0xb6, 0x13, 0xb, 0x77, 0xd, 0x6f, 0x5},
					Contract:   "",
				},
			},
			Tip:        common.NewAmount(1),
			GasLimit:   common.NewAmount(1),
			GasPrice:   common.NewAmount(30000),
			CreateTime: 0,
			Type:       transaction.TxTypeCoinbase,
		},
	}

	// create BlockChain
	bc := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, transactionpool.NewTransactionPool(nil, 100), 100)
	blk, err := bc.GetTailBlock()
	require.Nil(t, err)

	acc := account.NewAccount()
	bp := blockchain.NewBlockPool(nil)
	producerInfo := blockproducerinfo.NewBlockProducerInfo(acc.GetAddress().String())
	conss := consensus.NewDPOS(producerInfo)
	conss.SetDynasty(consensus.NewDynasty([]string{"dc6YApYeNS2MLyrKKvFVDYMGTy7RGQmRzm"}, 1, 99))
	signKey := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	conss.SetKey(signKey)
	bcm := NewBlockchainManager(bc, bp, nil, conss)

	// bad hash
	b1 := block.NewBlockWithRawInfo([]byte("invalid hash"), blk.GetHash(), 1, 0, 1, nil)
	assert.False(t, bcm.VerifyBlock(b1))

	// consensus validate failed
	b2 := block.NewBlockWithRawInfo([]byte{}, blk.GetHash(), 2, 0, 2, []*transaction.Transaction{})
	b2.SetHash(lblock.CalculateHash(b2))
	success := lblock.SignBlock(b2, signKey)
	assert.True(t, success)
	assert.False(t, bcm.VerifyBlock(b2))

	// valid block
	b3 := block.NewBlockWithRawInfo([]byte{}, blk.GetHash(), 3, 0, 3, txs)
	b3.SetHash(lblock.CalculateHash(b3))
	success = lblock.SignBlock(b3, signKey)
	assert.True(t, success)
	assert.True(t, bcm.VerifyBlock(b3))

	// double-minting (b4 has same timestamp as b3)
	b4 := block.NewBlockWithRawInfo([]byte{}, b3.GetHash(), 4, 0, 4, txs)
	b4.SetHash(lblock.CalculateHash(b4))
	success = lblock.SignBlock(b4, signKey)
	assert.True(t, success)
	assert.False(t, bcm.VerifyBlock(b4))
}

func TestBlockchainManager_RequestBlockHandler(t *testing.T) {

	// there are two blockchains, "sender" and "requester"
	// "sender" has a block, and "requester" does not
	// "requester" is the source of the RequestBlock command
	// "sender" handles the command, and sends the block to "requester"

	rfl := storage.NewRamFileLoader(confDir, "test.conf")
	defer rfl.DeleteFolder()

	senderNode := network.NewNode(rfl.File, nil)
	senderNode.Start(senderNodePort, "")
	defer senderNode.Stop()
	requesterNode := network.NewNode(rfl.File, nil)
	requesterNode.Start(requesterNodePort, "")
	defer requesterNode.Stop()
	senderNode.GetNetwork().ConnectToSeed(requesterNode.GetHostPeerInfo())
	// create BlockChains
	libPolicy := &blockchainMock.LIBPolicy{}
	libPolicy.On("GetMinConfirmationNum").Return(6)
	libPolicy.On("IsBypassingLibCheck").Return(true)
	bcSender := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), libPolicy, transactionpool.NewTransactionPool(senderNode, 100), 100)
	bcRequester := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), libPolicy, transactionpool.NewTransactionPool(requesterNode, 100), 100)
	genesis, err := bcSender.GetTailBlock()
	require.Nil(t, err)

	bp := blockchain.NewBlockPool(nil)

	acc := account.NewAccount()
	producerInfo := blockproducerinfo.NewBlockProducerInfo(acc.GetAddress().String())
	conss := consensus.NewDPOS(producerInfo)
	conss.SetDynasty(consensus.NewDynasty([]string{"dc6YApYeNS2MLyrKKvFVDYMGTy7RGQmRzm"}, 1, 1))
	signKey := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	conss.SetKey(signKey)
	bcmSender := NewBlockchainManager(bcSender, bp, senderNode, conss)
	bcmRequester := NewBlockchainManager(bcRequester, bp, requesterNode, conss)
	transaction.SetSubsidy(0)

	txs := []*transaction.Transaction{
		{
			ID: []byte{0x74},
			Vin: []transactionbase.TXInput{
				{
					Txid:      []byte{0x0},
					Vout:      0,
					Signature: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
					PubKey:    acc.GetKeyPair().GetPublicKey(),
				},
			},
			Vout: []transactionbase.TXOutput{
				{
					Value:      common.NewAmount(10),
					PubKeyHash: account.PubKeyHash{0x5a, 0xf8, 0xbf, 0x23, 0x39, 0x70, 0xf0, 0x9b, 0x65, 0x31, 0x98, 0xca, 0xed, 0x6c, 0xb6, 0x13, 0xb, 0x77, 0xd, 0x6f, 0x5},
					Contract:   "",
				},
			},
			Tip:        common.NewAmount(10),
			GasLimit:   common.NewAmount(1),
			GasPrice:   common.NewAmount(30000),
			CreateTime: 0,
			Type:       transaction.TxTypeCoinbase,
		},
	}
	blk := block.NewBlockWithRawInfo([]byte{}, genesis.GetHash(), 0, 0, 2, txs)
	blk.SetHash(lblock.CalculateHash(blk))
	ok := lblock.SignBlock(blk, signKey)
	assert.True(t, ok)
	bcmSender.blockchain.AddBlockContextToTail(PrepareBlockContext(bcSender, blk))
	// check that block was properly added to sender blockchain
	result, err := bcmSender.blockchain.GetBlockByHash(blk.GetHash())
	assert.Nil(t, err)
	if err == nil {
		assert.Equal(t, result.GetHeader(), blk.GetHeader())
		assert.Equal(t, len(result.GetTransactions()), len(blk.GetTransactions()))
		for i, tx := range result.GetTransactions() {
			assert.Equal(t, tx, blk.GetTransactions()[i])
		}
	}
	// check that requester blockchain does not have the block yet
	result, err = bcmRequester.blockchain.GetBlockByHash(blk.GetHash())
	assert.Nil(t, result)
	assert.Equal(t, ErrBlockDoesNotExist, err)

	// create and handle RequestBlock command
	requestBlock := &lblockchainpb.RequestBlock{Hash: blk.GetHash()}
	requestBlockBytes, err := proto.Marshal(requestBlock)
	assert.Nil(t, err)
	cmd := networkmodel.NewDappRcvdCmdContext(networkmodel.NewDappCmd(RequestBlock, requestBlockBytes, false), requesterNode.GetHostPeerInfo())
	time.Sleep(time.Millisecond * 500)
	bcmSender.RequestBlockHandler(cmd)
	util.WaitDoneOrTimeout(func() bool {
		_, err := bcmRequester.blockchain.GetBlockByHash(blk.GetHash())
		return err == nil
	}, 10)

	// check that the block was successfully sent to the requester
	result, err = bcmRequester.blockchain.GetBlockByHash(blk.GetHash())
	assert.Nil(t, err)
	if err == nil {
		assert.Equal(t, result.GetHeader(), blk.GetHeader())
		assert.Equal(t, len(result.GetTransactions()), len(blk.GetTransactions()))
		for i, tx := range result.GetTransactions() {
			assert.Equal(t, tx, blk.GetTransactions()[i])
		}
	}
}

func TestCopyAndRevertUtxos(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	coinbaseAddr := account.NewAddress("testaddress")
	bc := CreateBlockchain(coinbaseAddr, db, nil, transactionpool.NewTransactionPool(nil, 128000), 100000)

	blk1 := core.GenerateUtxoMockBlockWithoutInputs() // contains 2 UTXOs for address1
	blk2 := core.GenerateUtxoMockBlockWithInputs()    // contains tx that transfers address1's UTXOs to address2 with a change

	bc.AddBlockContextToTail(PrepareBlockContext(bc, blk1))
	bc.AddBlockContextToTail(PrepareBlockContext(bc, blk2))

	utxoIndex := lutxo.NewUTXOIndex(bc.GetUtxoCache())

	var address1Bytes = []byte("address1000000000000000000000000")
	var address2Bytes = []byte("address2000000000000000000000000")
	var ta1 = account.NewTransactionAccountByPubKey(address1Bytes)
	var ta2 = account.NewTransactionAccountByPubKey(address2Bytes)

	addr1UTXOs := utxoIndex.GetAllUTXOsByPubKeyHash([]byte(ta1.GetPubKeyHash()))
	addr2UTXOs := utxoIndex.GetAllUTXOsByPubKeyHash([]byte(ta2.GetPubKeyHash()))
	// Expect address1 to have 1 utxo of $4
	assert.Equal(t, 1, addr1UTXOs.Size())
	utxo1 := addr1UTXOs.GetAllUtxos()[0]
	assert.Equal(t, common.NewAmount(4), utxo1.Value)

	// Expect address2 to have 2 utxos totaling $8
	assert.Equal(t, 2, addr2UTXOs.Size())

	// Rollback to blk1, address1 has a $5 utxo and a $7 utxo, total $12, and address2 has nothing
	indexSnapshot, _, err := RevertUtxoAndScStateAtBlockHash(db, bc, blk1.GetHash())
	if err != nil {
		panic(err)
	}

	addr1UtxoTx := indexSnapshot.GetAllUTXOsByPubKeyHash(ta1.GetPubKeyHash())
	assert.Equal(t, 2, addr1UtxoTx.Size())

	tx1 := core.MockUtxoTransactionWithoutInputs()

	assert.Equal(t, common.NewAmount(5), addr1UtxoTx.GetUtxo(tx1.ID, 0).Value)
	assert.Equal(t, common.NewAmount(7), addr1UtxoTx.GetUtxo(tx1.ID, 1).Value)
	assert.Equal(t, 0, indexSnapshot.GetAllUTXOsByPubKeyHash(ta2.GetPubKeyHash()).Size())
}
