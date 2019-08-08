package block_logic

import (
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

func TestHashTransactions(t *testing.T) {

	var parentBlk = block.NewBlockWithRawInfo(
		[]byte{'a'},
		[]byte{'e', 'c'},
		0,
		time.Now().Unix(),
		0,
		nil,
	)

	var expectHash = []uint8([]byte{0x5d, 0xf6, 0xe0, 0xe2, 0x76, 0x13, 0x59, 0xd3, 0xa, 0x82, 0x75, 0x5, 0x8e, 0x29, 0x9f, 0xcc, 0x3, 0x81, 0x53, 0x45, 0x45, 0xf5, 0x5c, 0xf4, 0x3e, 0x41, 0x98, 0x3f, 0x5d, 0x4c, 0x94, 0x56})

	blk := block.NewBlock([]*core.Transaction{{}}, parentBlk, "")
	hash := HashTransactions(blk)
	assert.Equal(t, expectHash, hash)
}

func TestBlock_VerifyHash(t *testing.T) {
	b1 := core.GenerateMockBlock()

	//The mocked block does not have correct hash Value
	assert.False(t, VerifyHash(b1))

	//calculate correct hash Value
	hash := CalculateHash(b1)
	b1.SetHash(hash)
	assert.True(t, VerifyHash(b1))

	//calculate a hash Value with a different nonce
	b1.SetNonce(b1.GetNonce() + 1)
	hash = CalculateHashWithNonce(b1)
	b1.SetHash(hash)
	assert.False(t, VerifyHash(b1))

	hash = CalculateHashWithoutNonce(b1)
	b1.SetHash(hash)
	assert.False(t, VerifyHash(b1))
}

func TestCalculateHashWithNonce(t *testing.T) {
	var parentBlk = block.NewBlockWithRawInfo(
		[]byte{'a'},
		[]byte{'e', 'c'},
		0,
		0,
		0,
		nil,
	)

	blk := block.NewBlock([]*core.Transaction{{}}, parentBlk, "")
	blk.SetTimestamp(0)
	expectHash1 := hash.Hash{0x3f, 0x2f, 0xec, 0xb4, 0x33, 0xf0, 0xd1, 0x1a, 0xa6, 0xf4, 0xf, 0xb8, 0x7f, 0x8f, 0x99, 0x11, 0xae, 0xe7, 0x42, 0xf4, 0x69, 0x7d, 0xf1, 0xaa, 0xc8, 0xd0, 0xfc, 0x40, 0xa2, 0xd8, 0xb1, 0xa5}
	blk.SetNonce(1)
	assert.Equal(t, hash.Hash(expectHash1), CalculateHashWithNonce(blk))
	expectHash2 := hash.Hash{0xe7, 0x57, 0x13, 0xc6, 0x8a, 0x98, 0x58, 0xb3, 0x5, 0x70, 0x6e, 0x33, 0xf0, 0x95, 0xd8, 0x1a, 0xbc, 0x76, 0xef, 0x30, 0x14, 0x59, 0x88, 0x11, 0x3c, 0x11, 0x59, 0x92, 0x65, 0xd5, 0xd3, 0x4c}
	blk.SetNonce(2)
	assert.Equal(t, hash.Hash(expectHash2), CalculateHashWithNonce(blk))
}

func TestBlock_VerifyTransactions(t *testing.T) {
	// Prepare test data
	// Padding Address to 32 Byte
	var address1Bytes = []byte("address1000000000000000000000000")
	var address1Hash, _ = account.NewUserPubKeyHash(address1Bytes)

	normalCoinbaseTX := core.NewCoinbaseTX(address1Hash.GenerateAddress(), "", 1, common.NewAmount(0))
	rewardTX := core.NewRewardTx(1, map[string]string{address1Hash.GenerateAddress().String(): "10"})
	userPubKey := account.NewKeyPair().GetPublicKey()
	userPubKeyHash, _ := account.NewUserPubKeyHash(userPubKey)
	userAddr := userPubKeyHash.GenerateAddress()
	contractPubKeyHash := account.NewContractPubKeyHash()
	contractAddr := contractPubKeyHash.GenerateAddress()

	txIdStr := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	generatedTxId, err := hex.DecodeString(txIdStr)
	assert.Nil(t, err)
	fmt.Println(hex.EncodeToString(generatedTxId))
	generatedTX := &core.Transaction{
		generatedTxId,
		[]core.TXInput{
			{[]byte("prevtxid"), 0, []byte("txid"), []byte(contractPubKeyHash)},
			{[]byte("prevtxid"), 1, []byte("txid"), []byte(contractPubKeyHash)},
		},
		[]core.TXOutput{
			*core.NewTxOut(common.NewAmount(23), userAddr, ""),
			*core.NewTxOut(common.NewAmount(10), contractAddr, ""),
		},
		common.NewAmount(7),
		common.NewAmount(0),
		common.NewAmount(0),
	}

	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = account.GenerateKeyPairByPrivateKey(prikey1).GetPublicKey()
	var pkHash1, _ = account.NewUserPubKeyHash(pubkey1)
	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var pubkey2 = account.GenerateKeyPairByPrivateKey(prikey2).GetPublicKey()
	var pkHash2, _ = account.NewUserPubKeyHash(pubkey2)

	dependentTx1 := core.NewTransactionByVin(util.GenerateRandomAoB(1), 1, pubkey1, 10, pkHash2, 3)
	dependentTx2 := core.NewTransactionByVin(dependentTx1.ID, 0, pubkey2, 5, pkHash1, 5)
	dependentTx3 := core.NewTransactionByVin(dependentTx2.ID, 0, pubkey1, 1, pkHash2, 4)

	tx2Utxo1 := core.UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, core.UtxoNormal}

	tx1Utxos := map[string][]*core.UTXO{
		pkHash2.String(): {&core.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, core.UtxoNormal}},
	}
	dependentTx2.Sign(account.GenerateKeyPairByPrivateKey(prikey2).GetPrivateKey(), tx1Utxos[pkHash2.String()])
	dependentTx3.Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), []*core.UTXO{&tx2Utxo1})

	tests := []struct {
		name  string
		txs   []*core.Transaction
		utxos map[string][]*core.UTXO
		ok    bool
	}{
		{
			"normal txs",
			[]*core.Transaction{&normalCoinbaseTX},
			map[string][]*core.UTXO{},
			true,
		},
		{
			"no txs",
			[]*core.Transaction{},
			make(map[string][]*core.UTXO),
			true,
		},
		{
			"invalid normal txs",
			[]*core.Transaction{{
				ID: []byte("txid"),
				Vin: []core.TXInput{{
					[]byte("tx1"),
					0,
					util.GenerateRandomAoB(2),
					address1Bytes,
				}},
				Vout: core.MockUtxoOutputsWithInputs(),
				Tip:  common.NewAmount(5),
			}},
			map[string][]*core.UTXO{},
			false,
		},
		{
			"normal dependent txs",
			[]*core.Transaction{&dependentTx2, &dependentTx3},
			tx1Utxos,
			true,
		},
		{
			"invalid dependent txs",
			[]*core.Transaction{&dependentTx3, &dependentTx2},
			tx1Utxos,
			false,
		},
		{
			"reward tx",
			[]*core.Transaction{&rewardTX},
			map[string][]*core.UTXO{
				contractPubKeyHash.String(): {
					{*core.NewTXOutput(common.NewAmount(0), contractAddr), []byte("prevtxid"), 0, core.UtxoNormal},
				},
				userPubKeyHash.String(): {
					{*core.NewTXOutput(common.NewAmount(1), userAddr), []byte("txinid"), 0, core.UtxoNormal},
				},
			},
			false,
		},
		{
			"generated tx",
			[]*core.Transaction{generatedTX},
			map[string][]*core.UTXO{
				contractPubKeyHash.String(): {
					{*core.NewTXOutput(common.NewAmount(20), contractAddr), []byte("prevtxid"), 0, core.UtxoNormal},
					{*core.NewTXOutput(common.NewAmount(20), contractAddr), []byte("prevtxid"), 1, core.UtxoNormal},
				},
				userPubKeyHash.String(): {
					{*core.NewTXOutput(common.NewAmount(1), userAddr), []byte("txinid"), 0, core.UtxoNormal},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := storage.NewRamStorage()
			index := make(map[string]*core.UTXOTx)

			for key, addrUtxos := range tt.utxos {
				utxoTx := core.NewUTXOTx()
				for _, addrUtxo := range addrUtxos {
					utxoTx.PutUtxo(addrUtxo)
				}
				index[key] = &utxoTx
			}

			utxoIndex := core.UTXOIndex{index, core.NewUTXOCache(db), &sync.RWMutex{}}
			scState := core.NewScState()
			var parentBlk = block.NewBlockWithRawInfo(
				[]byte{'a'},
				[]byte{'e', 'c'},
				0,
				time.Now().Unix(),
				0,
				nil,
			)
			blk := block.NewBlock(tt.txs, parentBlk, "")
			assert.Equal(t, tt.ok, VerifyTransactions(blk, &utxoIndex, scState, nil, parentBlk))
		})
	}
}
