package lblock

import (
	"encoding/hex"
	"fmt"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"testing"
	"time"

	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/lutxo"

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

	blk := block.NewBlock([]*transaction.Transaction{{}}, parentBlk, "")
	hash := HashTransactions(blk)
	assert.Equal(t, expectHash, hash)
}

func TestBlock_VerifyHash(t *testing.T) {
	b1 := block.GenerateMockBlock()

	//The mocked block does not have correct h Value
	assert.False(t, VerifyHash(b1))

	//calculate correct h Value
	h := CalculateHash(b1)
	b1.SetHash(h)
	assert.True(t, VerifyHash(b1))

	//calculate a h Value with a different nonce
	b1.SetNonce(b1.GetNonce() + 1)
	h = CalculateHashWithNonce(b1)
	assert.False(t, VerifyHash(b1))

	h = CalculateHashWithoutNonce(b1)
	b1.SetHash(h)
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

	blk := block.NewBlock([]*transaction.Transaction{{}}, parentBlk, "")
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
	var address1TA = account.NewTransactionAccountByPubKey(address1Bytes)

	rewardTX := ltransaction.NewRewardTx(1, map[string]string{address1TA.GetAddress().String(): "10"})
	userPubKey := account.NewKeyPair().GetPublicKey()
	userTA := account.NewTransactionAccountByPubKey(userPubKey)
	contractTA := account.NewContractTransactionAccount()

	txIdStr := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	generatedTxId, err := hex.DecodeString(txIdStr)
	assert.Nil(t, err)
	fmt.Println(hex.EncodeToString(generatedTxId))
	generatedTX := &transaction.Transaction{
		generatedTxId,
		[]transactionbase.TXInput{
			{[]byte("prevtxid"), 0, []byte("txid"), []byte(contractTA.GetPubKeyHash())},
			{[]byte("prevtxid"), 1, []byte("txid"), []byte(contractTA.GetPubKeyHash())},
		},
		[]transactionbase.TXOutput{
			*transactionbase.NewTxOut(common.NewAmount(23), userTA, ""),
			*transactionbase.NewTxOut(common.NewAmount(10), contractTA, ""),
		},
		common.NewAmount(7),
		common.NewAmount(0),
		common.NewAmount(0),
		0,
		transaction.TxTypeNormal,
	}

	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var ta1 = account.NewAccountByPrivateKey(prikey1)

	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var ta2 = account.NewAccountByPrivateKey(prikey2)

	dependentTx1 := NewTransactionByVin(util.GenerateRandomAoB(1), 1, ta1.GetKeyPair().GetPublicKey(), 10, ta2.GetPubKeyHash(), 3)
	dependentTx2 := NewTransactionByVin(dependentTx1.ID, 0, ta2.GetKeyPair().GetPublicKey(), 5, ta1.GetPubKeyHash(), 5)
	dependentTx3 := NewTransactionByVin(dependentTx2.ID, 0, ta1.GetKeyPair().GetPublicKey(), 1, ta2.GetPubKeyHash(), 4)

	tx2Utxo1 := utxo.UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}

	tx1Utxos := map[string][]*utxo.UTXO{
		ta2.GetPubKeyHash().String(): {&utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}}},
	}
	ltransaction.NewTxDecorator(&dependentTx2).Sign(account.GenerateKeyPairByPrivateKey(prikey2).GetPrivateKey(), tx1Utxos[ta2.GetPubKeyHash().String()])
	ltransaction.NewTxDecorator(&dependentTx3).Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo1})

	tests := []struct {
		name  string
		txs   []*transaction.Transaction
		utxos map[string][]*utxo.UTXO
		ok    bool
	}{
		{
			"no txs",
			[]*transaction.Transaction{},
			make(map[string][]*utxo.UTXO),
			true,
		},
		{
			"invalid normal txs",
			[]*transaction.Transaction{{
				ID: []byte("txid"),
				Vin: []transactionbase.TXInput{{
					[]byte("tx1"),
					0,
					util.GenerateRandomAoB(2),
					address1Bytes,
				}},
				Vout: core.MockUtxoOutputsWithInputs(),
				Tip:  common.NewAmount(5),
			}},
			map[string][]*utxo.UTXO{},
			false,
		},
		{
			"normal dependent txs",
			[]*transaction.Transaction{&dependentTx2, &dependentTx3},
			tx1Utxos,
			true,
		},
		{
			"invalid dependent txs",
			[]*transaction.Transaction{&dependentTx3, &dependentTx2},
			tx1Utxos,
			false,
		},
		{
			"reward tx",
			[]*transaction.Transaction{&rewardTX},
			map[string][]*utxo.UTXO{
				contractTA.GetPubKeyHash().String(): {
					{*transactionbase.NewTXOutput(common.NewAmount(0), contractTA), []byte("prevtxid"), 0, utxo.UtxoNormal, []byte{}, []byte{}},
				},
				userTA.GetPubKeyHash().String(): {
					{*transactionbase.NewTXOutput(common.NewAmount(1), userTA), []byte("txinid"), 0, utxo.UtxoNormal, []byte{}, []byte{}},
				},
			},
			false,
		},
		{
			"generated tx",
			[]*transaction.Transaction{generatedTX},
			map[string][]*utxo.UTXO{
				contractTA.GetPubKeyHash().String(): {
					{*transactionbase.NewTXOutput(common.NewAmount(20), contractTA), []byte("prevtxid"), 0, utxo.UtxoNormal, []byte{}, []byte{}},
					{*transactionbase.NewTXOutput(common.NewAmount(20), contractTA), []byte("prevtxid"), 1, utxo.UtxoNormal, []byte{}, []byte{}},
				},
				userTA.GetPubKeyHash().String(): {
					{*transactionbase.NewTXOutput(common.NewAmount(1), userTA), []byte("txinid"), 0, utxo.UtxoNormal, []byte{}, []byte{}},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := storage.NewRamStorage()
			index := make(map[string]*utxo.UTXOTx)

			for key, addrUtxos := range tt.utxos {
				utxoTx := utxo.NewUTXOTx()
				for _, addrUtxo := range addrUtxos {
					utxoTx.PutUtxo(addrUtxo)
				}
				index[key] = &utxoTx
			}

			cache := utxo.NewUTXOCache(db)
			utxoIndex := lutxo.NewUTXOIndex(cache)
			utxoIndex.SetIndexAdd(index)
			//{index, utxo.NewUTXOCache(db), &sync.RWMutex{}}
			scState := scState.NewScState(cache)
			var parentBlk = block.NewBlockWithRawInfo(
				[]byte{'a'},
				[]byte{'e', 'c'},
				0,
				time.Now().Unix(),
				0,
				nil,
			)
			// add coinbase
			totalTip := common.NewAmount(0)
			for _, tx := range tt.txs {
				totalTip = totalTip.Add(tx.Tip)
			}
			coninbaseTx := ltransaction.NewCoinbaseTX(address1TA.GetAddress(), "", parentBlk.GetHeight()+1, totalTip)
			tt.txs = append(tt.txs, &coninbaseTx)
			blk := block.NewBlock(tt.txs, parentBlk, "")
			assert.Equal(t, tt.ok, VerifyTransactions(blk, utxoIndex, scState, parentBlk, db))
		})
	}
}

func TestGenerateSignature(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		data        hash.Hash
		expectedRes hash.Hash
		expectedErr interface{}
	}{
		{
			name:        "successful",
			key:         "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71",
			data:        hash.Hash{0x57, 0x78, 0xcb, 0xa8, 0x30, 0x31, 0x26, 0x57, 0xba, 0x94, 0x85, 0x62, 0x78, 0xf, 0x25, 0xbc, 0x84, 0x2b, 0x1a, 0x38, 0xbc, 0x58, 0xdf, 0x2, 0xa1, 0xbe, 0x3b, 0x92, 0x9f, 0x81, 0x99, 0xa6},
			expectedRes: hash.Hash{0xe9, 0x4d, 0x87, 0x72, 0xd6, 0xfd, 0x65, 0x7a, 0xf, 0xef, 0xe7, 0xd, 0x9f, 0xae, 0x2b, 0xc7, 0x95, 0xa2, 0xe6, 0xd, 0xd4, 0xd5, 0x6d, 0xc2, 0x62, 0xa, 0x58, 0x88, 0xd3, 0x13, 0xfc, 0xec, 0x22, 0x58, 0x86, 0xd3, 0xef, 0x97, 0x1c, 0x62, 0x9e, 0x80, 0x25, 0x2c, 0x10, 0x1f, 0x98, 0xa6, 0x4f, 0x9e, 0xdf, 0x8e, 0xce, 0x83, 0x14, 0xb8, 0xa5, 0xe2, 0x1, 0x88, 0x68, 0x6a, 0x14, 0xea, 0x1},
			expectedErr: nil,
		},
		{
			name:        "invalid key",
			key:         "invalid",
			data:        hash.Hash{0x57, 0x78, 0xcb, 0xa8, 0x30, 0x31, 0x26, 0x57, 0xba, 0x94, 0x85, 0x62, 0x78, 0xf, 0x25, 0xbc, 0x84, 0x2b, 0x1a, 0x38, 0xbc, 0x58, 0xdf, 0x2, 0xa1, 0xbe, 0x3b, 0x92, 0x9f, 0x81, 0x99, 0xa6},
			expectedRes: hash.Hash{},
			expectedErr: hex.InvalidByteError(0x69),
		},
		{
			name:        "invalid input hash length",
			key:         "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71",
			data:        hash.Hash{0xde, 0xad, 0xbe, 0xef},
			expectedRes: hash.Hash{},
			expectedErr: secp256k1.ErrInvalidMsgLen,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateSignature(tt.key, tt.data)
			assert.Equal(t, tt.expectedRes, result)
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestSignBlock(t *testing.T) {
	b1 := block.GenerateMockBlock()
	b1Hash := CalculateHash(b1)
	b1.SetHash(b1Hash)
	key := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	expectedSign, _ := generateSignature(key, b1Hash)

	ok := SignBlock(b1, "")
	assert.False(t, ok)
	assert.Nil(t, b1.GetSign())

	ok = SignBlock(b1, "invalid")
	assert.False(t, ok)
	assert.Nil(t, b1.GetSign())

	ok = SignBlock(b1, key)
	assert.True(t, ok)
	assert.Equal(t, expectedSign, b1.GetSign())

	fmt.Printf("%#v\n", b1Hash)
}

func TestVerifyGasTxs(t *testing.T) {
	gasRewardTx1 := &transaction.Transaction{
		ID:  util.GenerateRandomAoB(1),
		Vin: transactionbase.GenerateFakeTxInputs(),
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(10), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
		},
		Type: transaction.TxTypeGasReward,
	}
	gasRewardTx2 := &transaction.Transaction{
		ID:  util.GenerateRandomAoB(1),
		Vin: transactionbase.GenerateFakeTxInputs(),
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(5), PubKeyHash: account.PubKeyHash([]byte{0xc7, 0x48}), Contract: "test"},
		},
		Type: transaction.TxTypeGasReward,
	}
	gasChangeTx1 := &transaction.Transaction{
		ID:  util.GenerateRandomAoB(1),
		Vin: transactionbase.GenerateFakeTxInputs(),
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash([]byte{0xc8, 0x47}), Contract: "test"},
		},
		Type: transaction.TxTypeGasChange,
	}
	gasChangeTx2 := &transaction.Transaction{
		ID:  util.GenerateRandomAoB(1),
		Vin: transactionbase.GenerateFakeTxInputs(),
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(3), PubKeyHash: account.PubKeyHash([]byte{0xc9, 0x46}), Contract: "test"},
		},
		Type: transaction.TxTypeGasChange,
	}

	tests := []struct {
		name          string
		blockTxs      []*transaction.Transaction
		totalGasFee   *common.Amount
		actualGasList []uint64
		expectedRes   bool
	}{
		{
			name:          "valid GasReward only",
			blockTxs:      []*transaction.Transaction{gasRewardTx1, gasRewardTx2},
			totalGasFee:   common.NewAmount(15),
			actualGasList: []uint64{10, 5},
			expectedRes:   true,
		},
		{
			name:          "valid GasChange only",
			blockTxs:      []*transaction.Transaction{gasChangeTx1, gasChangeTx2},
			totalGasFee:   common.NewAmount(5),
			actualGasList: []uint64{},
			expectedRes:   true,
		},
		{
			name:          "valid GasReward and GasChange",
			blockTxs:      []*transaction.Transaction{gasRewardTx1, gasChangeTx1, gasRewardTx2, gasChangeTx2},
			totalGasFee:   common.NewAmount(20),
			actualGasList: []uint64{10, 5},
			expectedRes:   true,
		},
		{
			name:          "zero totalGasFee and empty actualGasList",
			blockTxs:      []*transaction.Transaction{gasRewardTx1, gasRewardTx2},
			totalGasFee:   common.NewAmount(0),
			actualGasList: []uint64{},
			expectedRes:   true,
		},
		{
			name:          "gasReward amount not found in actualGasList",
			blockTxs:      []*transaction.Transaction{gasRewardTx1, gasRewardTx2},
			totalGasFee:   common.NewAmount(15),
			actualGasList: []uint64{10},
			expectedRes:   false,
		},
		{
			name:          "totalGasFee too high",
			blockTxs:      []*transaction.Transaction{gasChangeTx1, gasChangeTx2},
			totalGasFee:   common.NewAmount(100),
			actualGasList: []uint64{},
			expectedRes:   false,
		},
		{
			name:          "totalGasFee too low",
			blockTxs:      []*transaction.Transaction{gasRewardTx1, gasRewardTx2},
			totalGasFee:   common.NewAmount(1),
			actualGasList: []uint64{10, 5},
			expectedRes:   false,
		},
		{
			name: "GetRewardValue failed",
			blockTxs: []*transaction.Transaction{
				{
					ID:   util.GenerateRandomAoB(1),
					Vin:  transactionbase.GenerateFakeTxInputs(),
					Vout: []transactionbase.TXOutput{},
					Type: transaction.TxTypeGasReward,
				},
			},
			totalGasFee:   common.NewAmount(10),
			actualGasList: []uint64{10},
			expectedRes:   false,
		},
		{
			name: "GetChangeValue failed",
			blockTxs: []*transaction.Transaction{
				{
					ID:   util.GenerateRandomAoB(1),
					Vin:  transactionbase.GenerateFakeTxInputs(),
					Vout: []transactionbase.TXOutput{},
					Type: transaction.TxTypeGasChange,
				},
			},
			totalGasFee:   common.NewAmount(10),
			actualGasList: []uint64{10},
			expectedRes:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyGasTxs(tt.blockTxs, tt.totalGasFee, tt.actualGasList)
			assert.Equal(t, tt.expectedRes, result)
		})
	}
}

func NewTransactionByVin(vinTxId []byte, vinVout int, vinPubkey []byte, voutValue uint64, voutPubKeyHash account.PubKeyHash, tip uint64) transaction.Transaction {
	tx := transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{vinTxId, vinVout, nil, vinPubkey},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(voutValue), voutPubKeyHash, ""},
		},
		Tip:  common.NewAmount(tip),
		Type: transaction.TxTypeNormal,
	}
	tx.ID = tx.Hash()
	return tx
}
