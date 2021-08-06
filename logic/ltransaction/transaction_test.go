package ltransaction

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var tx1 = transaction.Transaction{
	ID:       util.GenerateRandomAoB(1),
	Vin:      transactionbase.GenerateFakeTxInputs(),
	Vout:     transactionbase.GenerateFakeTxOutputs(),
	Tip:      common.NewAmount(5),
	GasLimit: common.NewAmount(0),
	GasPrice: common.NewAmount(0),
}

func TestSign(t *testing.T) {
	// Fake a key pair
	privKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("fakefakefakefakefakefakefakefakefakefake")))
	ecdsaPubKey, _ := secp256k1.FromECDSAPublicKey(&privKey.PublicKey)
	pubKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	ta := account.NewTransactionAccountByPubKey(pubKey)

	// Previous transactions containing UTXO of the Address
	prevTXs := []*utxo.UTXO{
		{transactionbase.TXOutput{common.NewAmount(13), ta.GetPubKeyHash(), ""}, []byte("01"), 0, utxo.UtxoNormal, []byte{}, []byte{}},
		{transactionbase.TXOutput{common.NewAmount(13), ta.GetPubKeyHash(), ""}, []byte("02"), 0, utxo.UtxoNormal, []byte{}, []byte{}},
		{transactionbase.TXOutput{common.NewAmount(13), ta.GetPubKeyHash(), ""}, []byte("03"), 0, utxo.UtxoNormal, []byte{}, []byte{}},
	}

	// New transaction to be signed (paid from the fake account)
	txin := []transactionbase.TXInput{
		{[]byte{1}, 0, nil, pubKey},
		{[]byte{3}, 0, nil, pubKey},
		{[]byte{3}, 2, nil, pubKey},
	}
	txout := []transactionbase.TXOutput{
		{common.NewAmount(19), ta.GetPubKeyHash(), ""},
	}
	tx := &transaction.Transaction{nil, txin, txout, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}

	// ltransaction.Sign the transaction
	err := NewTxDecorator(tx).Sign(*privKey, prevTXs)
	if assert.Nil(t, err) {
		// Assert that the signatures were created by the fake key pair
		for i, vin := range tx.Vin {

			if assert.NotNil(t, vin.Signature) {
				txCopy := tx.TrimmedCopy(false)
				txCopy.Vin[i].Signature = nil
				txCopy.Vin[i].PubKey = []byte(ta.GetPubKeyHash())

				verified, err := secp256k1.Verify(txCopy.Hash(), vin.Signature, ecdsaPubKey)
				assert.Nil(t, err)
				assert.True(t, verified)
			}
		}
	}
}

func TestTxNormal_Verify(t *testing.T) {
	acc := account.NewAccount()
	tx := &transaction.Transaction{
		ID: []byte{},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0x20, 0x21}, Vout: 0, Signature: nil, PubKey: acc.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
			{Value: common.NewAmount(20), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
		},
		Tip:      common.NewAmount(1),
		GasLimit: common.NewAmount(3000),
		GasPrice: common.NewAmount(2),
		Type:     transaction.TxTypeNormal,
	}
	vinUTXO := &utxo.UTXO{
		TXOutput: transactionbase.TXOutput{Value: common.NewAmount(6031), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
		Txid:     []byte{0x20, 0x21},
		TxIndex:  0,
		UtxoType: 0,
	}
	txCopy := tx.TrimmedCopy(true)
	tx.ID = (&txCopy).Hash()
	err := tx.Sign(acc.GetKeyPair().GetPrivateKey(), []*utxo.UTXO{vinUTXO})
	assert.Nil(t, err)

	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	// try to verify with vinUTXO not found in utxoIndex
	normalTx := &TxNormal{tx}
	err = normalTx.Verify(utxoIndex, 1)
	assert.Equal(t, transaction.ErrTXInputNotFound, err)

	utxoIndex.AddUTXO(vinUTXO.TXOutput, vinUTXO.Txid, vinUTXO.TxIndex)

	// successful verify
	err = normalTx.Verify(utxoIndex, 1)
	assert.Nil(t, err)
}

func TestVerifyCoinbaseTransaction(t *testing.T) {
	var prevTXs = map[string]transaction.Transaction{}

	var tx1 = transaction.Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  transactionbase.GenerateFakeTxInputs(),
		Vout: transactionbase.GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2),
	}

	var tx2 = transaction.Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  transactionbase.GenerateFakeTxInputs(),
		Vout: transactionbase.GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(5),
	}
	var tx3 = transaction.Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  transactionbase.GenerateFakeTxInputs(),
		Vout: transactionbase.GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(10),
	}
	var tx4 = transaction.Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  transactionbase.GenerateFakeTxInputs(),
		Vout: transactionbase.GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(20),
	}
	prevTXs[string(tx1.ID)] = tx2
	prevTXs[string(tx2.ID)] = tx3
	prevTXs[string(tx3.ID)] = tx4

	// test verifying coinbase transactions
	var t5 = NewCoinbaseTX(account.NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F"), "", 5, common.NewAmount(0))
	bh1 := make([]byte, 8)
	binary.BigEndian.PutUint64(bh1, 5)
	txin1 := transactionbase.TXInput{nil, -1, bh1, []byte("Reward to test")}
	txout1 := transactionbase.NewTXOutput(transaction.Subsidy, account.NewTransactionAccountByAddress(account.NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F")))
	var t6 = transaction.Transaction{nil, []transactionbase.TXInput{txin1}, []transactionbase.TXOutput{*txout1}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeCoinbase}

	// test valid coinbase transaction
	err5 := VerifyTransaction(&lutxo.UTXOIndex{}, &t5, 5)
	assert.Nil(t, err5)
	err6 := VerifyTransaction(&lutxo.UTXOIndex{}, &t6, 5)
	assert.Nil(t, err6)

	// test coinbase transaction with incorrect blockHeight
	err5 = VerifyTransaction(&lutxo.UTXOIndex{}, &t5, 10)
	assert.NotNil(t, err5)

	// test coinbase transaction with incorrect Subsidy
	bh2 := make([]byte, 8)
	binary.BigEndian.PutUint64(bh2, 5)
	txin2 := transactionbase.TXInput{nil, -1, bh2, []byte(nil)}
	txout2 := transactionbase.NewTXOutput(common.NewAmount(9), account.NewTransactionAccountByAddress(account.NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F")))
	var t7 = transaction.Transaction{nil, []transactionbase.TXInput{txin2}, []transactionbase.TXOutput{*txout2}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeCoinbase}
	err7 := VerifyTransaction(&lutxo.UTXOIndex{}, &t7, 5)
	assert.NotNil(t, err7)

}

func TestVerifyNoCoinbaseTransaction(t *testing.T) {
	// Fake a key pair
	privKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("fakefakefakefakefakefakefakefakefakefake")))
	privKeyByte, _ := secp256k1.FromECDSAPrivateKey(privKey)
	pubKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	ta := account.NewTransactionAccountByPubKey(pubKey)

	// Fake a wrong key pair
	wrongPrivKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("FAKEfakefakefakefakefakefakefakefakefake")))
	wrongPrivKeyByte, _ := secp256k1.FromECDSAPrivateKey(wrongPrivKey)
	wrongPubKey := append(wrongPrivKey.PublicKey.X.Bytes(), wrongPrivKey.PublicKey.Y.Bytes()...)
	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	utxoTx := utxo.NewUTXOTx()

	utxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(4), ta.GetPubKeyHash(), ""}, []byte{1}, 0, utxo.UtxoNormal, []byte{}, []byte{}})
	utxoTx.PutUtxo(&utxo.UTXO{transactionbase.TXOutput{common.NewAmount(3), ta.GetPubKeyHash(), ""}, []byte{2}, 1, utxo.UtxoNormal, []byte{}, []byte{}})

	utxoIndex.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta.GetPubKeyHash().String(): &utxoTx,
	})

	// Prepare a transaction to be verified
	txin := []transactionbase.TXInput{{[]byte{1}, 0, nil, pubKey}}
	txin1 := append(txin, transactionbase.TXInput{[]byte{2}, 1, nil, pubKey})      // Normal test
	txin2 := append(txin, transactionbase.TXInput{[]byte{2}, 1, nil, wrongPubKey}) // previous not found with wrong pubkey
	txin3 := append(txin, transactionbase.TXInput{[]byte{3}, 1, nil, pubKey})      // previous not found with wrong Txid
	txin4 := append(txin, transactionbase.TXInput{[]byte{2}, 2, nil, pubKey})      // previous not found with wrong TxIndex
	txout := []transactionbase.TXOutput{{common.NewAmount(7), ta.GetPubKeyHash(), ""}}
	txout2 := []transactionbase.TXOutput{{common.NewAmount(8), ta.GetPubKeyHash(), ""}} //Vout amount > Vin amount

	tests := []struct {
		name     string
		tx       transaction.Transaction
		signWith []byte
		ok       error
	}{
		{"normal", transaction.Transaction{nil, txin1, txout, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}, privKeyByte, nil},
		{"previous tx not found with wrong pubkey", transaction.Transaction{nil, txin2, txout, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}, privKeyByte, transaction.ErrTXInputNotFound},
		{"previous tx not found with wrong Txid", transaction.Transaction{nil, txin3, txout, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}, privKeyByte, transaction.ErrTXInputNotFound},
		{"previous tx not found with wrong TxIndex", transaction.Transaction{nil, txin4, txout, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}, privKeyByte, transaction.ErrTXInputNotFound},
		{"ID invalid", transaction.Transaction{nil, txin1, txout2, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}, privKeyByte, errors.New("Transaction: ID is invalid")},
		{"ltransaction.Sign invalid", transaction.Transaction{nil, txin1, txout, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}, wrongPrivKeyByte, errors.New("Transaction: ID is invalid")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tx.ID = tt.tx.Hash()
			// Generate signatures for all tx inputs
			for i := range tt.tx.Vin {
				txCopy := tt.tx.TrimmedCopy(false)
				txCopy.Vin[i].Signature = nil
				txCopy.Vin[i].PubKey = []byte(ta.GetPubKeyHash())
				signature, _ := secp256k1.Sign(txCopy.Hash(), tt.signWith)
				tt.tx.Vin[i].Signature = signature
			}

			// Verify the signatures
			err := VerifyTransaction(utxoIndex, &tt.tx, 0)
			assert.Equal(t, tt.ok, err)
		})
	}
}

func TestInvalidExecutionTx(t *testing.T) {
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = account.GenerateKeyPairByPrivateKey(prikey1).GetPublicKey()
	var ta1 = account.NewTransactionAccountByPubKey(pubkey1)
	var deploymentTx = transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{tx1.ID, 1, nil, pubkey1},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta1.GetPubKeyHash(), "dapp_schedule"},
		},
		Tip: common.NewAmount(1),
	}
	deploymentTx.ID = deploymentTx.Hash()
	contractPubkeyHash := deploymentTx.Vout[0].PubKeyHash

	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	utxoTx := utxo.NewUTXOTx()

	utxoTx.PutUtxo(&utxo.UTXO{deploymentTx.Vout[0], deploymentTx.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}})
	utxoIndex.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx,
	})

	var executionTx = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{deploymentTx.ID, 0, nil, pubkey1},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(3), contractPubkeyHash, "execution"},
		},
		Tip:  common.NewAmount(2),
		Type: transaction.TxTypeNormal,
	}
	executionTx.ID = executionTx.Hash()
	NewTxDecorator(executionTx).Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), utxoIndex.GetAllUTXOsByPubKeyHash(ta1.GetPubKeyHash()).GetAllUtxos())

	err1 := VerifyTransaction(lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage())), executionTx, 0)
	err2 := VerifyTransaction(utxoIndex, executionTx, 0)
	assert.NotNil(t, err1)
	assert.Nil(t, err2)
}

// Test with invalid tip amount, total vout != total vin
func TestInvalidTipTx(t *testing.T) {
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = account.GenerateKeyPairByPrivateKey(prikey1).GetPublicKey()
	var ta1 = account.NewContractTransactionAccount()
	var deploymentTx = transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{tx1.ID, 1, nil, pubkey1},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(50000), ta1.GetPubKeyHash(), "dapp_schedule"},
		},
		Tip: common.NewAmount(1),
	}
	deploymentTx.ID = deploymentTx.Hash()
	contractPubkeyHash := deploymentTx.Vout[0].PubKeyHash

	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	utxoTx := utxo.NewUTXOTx()

	utxoTx.PutUtxo(&utxo.UTXO{deploymentTx.Vout[0], deploymentTx.ID, 0, utxo.UtxoNormal, []byte{}, []byte{}})
	utxoIndex.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta1.GetPubKeyHash().String(): &utxoTx,
	})

	var executionTx = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{deploymentTx.ID, 0, nil, pubkey1},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(19998), contractPubkeyHash, "execution"},
		},
		Tip:      common.NewAmount(1),
		GasLimit: common.NewAmount(30000),
		GasPrice: common.NewAmount(1),
		Type:     transaction.TxTypeContract,
	}
	executionTx.ID = executionTx.Hash()
	NewTxDecorator(executionTx).Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), utxoIndex.GetAllUTXOsByPubKeyHash(ta1.GetPubKeyHash()).GetAllUtxos())

	err := VerifyTransaction(utxoIndex, executionTx, 0)
	assert.NotNil(t, err)
}

func TestTransaction_Execute(t *testing.T) {

	tests := []struct {
		name              string
		scAddr            string
		toAddr            string
		expectContractRun bool
	}{
		{
			name:              "CallAContract",
			scAddr:            "cWDSCWqwYRM6jNiN83PuRGvtcDuPpzBcfb",
			toAddr:            "cWDSCWqwYRM6jNiN83PuRGvtcDuPpzBcfb",
			expectContractRun: true,
		},
		{
			name:              "CallAWrongContractAddr",
			scAddr:            "cWDSCWqwYRM6jNiN83PuRGvtcDuPpzBcfb",
			toAddr:            "cavQdWxvUQU1HhBg1d7zJFwhf31SUaQwop",
			expectContractRun: false,
		},
		{
			name:              "NoPreviousContract",
			scAddr:            "",
			toAddr:            "cavQdWxvUQU1HhBg1d7zJFwhf31SUaQwop",
			expectContractRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := new(MockScEngine)
			contract := "helloworld!"
			toAccount := account.NewTransactionAccountByAddress(account.NewAddress(tt.toAddr))
			scAccount := account.NewTransactionAccountByAddress(account.NewAddress(tt.scAddr))

			toPKH := toAccount.GetPubKeyHash()
			scPKH := scAccount.GetPubKeyHash()

			scUtxo := utxo.UTXO{
				TxIndex: 0,
				Txid:    nil,
				TXOutput: transactionbase.TXOutput{
					Value:      common.NewAmount(0),
					PubKeyHash: scPKH,
					Contract:   contract,
				},
			}
			tx := transaction.Transaction{
				Vout:     []transactionbase.TXOutput{{nil, toPKH, "{\"function\":\"record\",\"args\":[\"dEhFf5mWTSe67mbemZdK3WiJh8FcCayJqm\",\"4\"]}"}},
				GasLimit: common.NewAmount(0),
				GasPrice: common.NewAmount(0),
			}
			ctx := NewTxContract(&tx)

			db := storage.NewRamStorage()
			defer db.Close()
			cache := utxo.NewUTXOCache(db)
			index := lutxo.NewUTXOIndex(cache)
			if tt.scAddr != "" {
				index.AddUTXO(scUtxo.TXOutput, nil, 0)
			}
			err := index.Save()
			assert.Nil(t, err)

			if tt.expectContractRun {
				sc.On("ImportSourceCode", contract)
				sc.On("ImportLocalStorage", mock.Anything)
				sc.On("ImportContractAddr", mock.Anything)
				sc.On("ImportSourceTXID", mock.Anything)
				sc.On("ImportRewardStorage", mock.Anything)
				sc.On("ImportTransaction", mock.Anything)
				sc.On("ImportContractCreateUTXO", mock.Anything)
				sc.On("ImportPrevUtxos", mock.Anything)
				sc.On("GetGeneratedTXs").Return([]*transaction.Transaction{})
				sc.On("ImportCurrBlockHeight", mock.Anything)
				sc.On("ImportSeed", mock.Anything)
				sc.On("ImportUtxoIndex", mock.Anything)
				sc.On("Execute", mock.Anything, mock.Anything).Return("")
			}
			parentBlk := core.GenerateMockBlock()
			preUTXO, err := lutxo.FindVinUtxosInUtxoPool(index, ctx.Transaction)
			assert.Nil(t, err)

			isContractDeployed := ctx.IsContractDeployed(index)
			ctx.Execute(preUTXO, isContractDeployed, index, scState.NewScState(cache), nil, sc, 0, parentBlk)
			sc.AssertExpectations(t)
		})
	}
}

//test IsCoinBase function
func TestIsCoinBase(t *testing.T) {
	var tx1 = transaction.Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  transactionbase.GenerateFakeTxInputs(),
		Vout: transactionbase.GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2),
	}

	assert.False(t, tx1.IsCoinbase())

	t2 := NewCoinbaseTX(account.NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F"), "", 0, common.NewAmount(0))

	assert.True(t, t2.IsCoinbase())

}

func TestTransaction_IsRewardTx(t *testing.T) {
	tests := []struct {
		name        string
		tx          transaction.Transaction
		expectedRes bool
	}{
		{"normal", NewRewardTx(1, map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "9"}), true},
		{"no rewards", NewRewardTx(1, nil), true},
		{"coinbase", NewCoinbaseTX(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 5, common.NewAmount(0)), false},
		{"normal tx", *core.MockTransaction(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.tx.IsRewardTx())
		})
	}
}

func TestNewRewardTx(t *testing.T) {
	rewards := map[string]string{
		"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "8",
		"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa": "9",
	}
	tx := NewRewardTx(5, rewards)

	values := []*common.Amount{tx.Vout[0].Value, tx.Vout[1].Value}
	assert.Contains(t, values, common.NewAmount(8))
	assert.Contains(t, values, common.NewAmount(9))
}

func TestTxContract_GasCountOfTxBase(t *testing.T) {
	contractAcc := account.NewContractTransactionAccount()
	tx := &transaction.Transaction{
		ID:  []byte{0x88},
		Vin: []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(10),
				PubKeyHash: contractAcc.GetPubKeyHash(),
				Contract:   "",
			},
		},
		Tip:        common.NewAmount(1),
		GasLimit:   common.NewAmount(3000),
		GasPrice:   common.NewAmount(2),
		CreateTime: 0,
		Type:       transaction.TxTypeContract,
	}
	txContract := NewTxContract(tx)
	result, err := txContract.GasCountOfTxBase()
	assert.Equal(t, common.NewAmount(20000), result)
	assert.Nil(t, err)

	txContract.Vout[0].Contract = "0123456789abcdef"
	result, err = txContract.GasCountOfTxBase()
	assert.Equal(t, common.NewAmount(20016), result)
	assert.Nil(t, err)
}

func TestTxContract_VerifyGas(t *testing.T) {
	contractAcc := account.NewContractTransactionAccount()
	tx := &transaction.Transaction{
		ID:  []byte{0x88},
		Vin: []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(10),
				PubKeyHash: contractAcc.GetPubKeyHash(),
				Contract:   "0123456789abcdef",
			},
		},
		Tip:        common.NewAmount(1),
		GasLimit:   common.NewAmount(20000),
		GasPrice:   common.NewAmount(2),
		CreateTime: 0,
		Type:       transaction.TxTypeContract,
	}
	txContract := NewTxContract(tx)
	err := txContract.VerifyGas(common.NewAmount(2000))
	assert.Equal(t, transaction.ErrOutOfGasLimit, err)

	txContract.GasLimit = common.NewAmount(30000)
	err = txContract.VerifyGas(common.NewAmount(59999))
	assert.Equal(t, transaction.ErrInsufficientBalance, err)

	err = txContract.VerifyGas(common.NewAmount(60000))
	assert.Nil(t, err)
}

func TestTxContract_GetTotalBalance(t *testing.T) {
	contractAcc := account.NewContractTransactionAccount()
	tx := &transaction.Transaction{
		ID:  []byte{0x88},
		Vin: []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(10),
				PubKeyHash: contractAcc.GetPubKeyHash(),
				Contract:   "0123456789abcdef",
			},
			{
				Value:      common.NewAmount(5),
				PubKeyHash: contractAcc.GetPubKeyHash(),
				Contract:   "",
			},
		},
		Tip:        common.NewAmount(1),
		GasLimit:   common.NewAmount(20000),
		GasPrice:   common.NewAmount(2),
		CreateTime: 0,
		Type:       transaction.TxTypeContract,
	}
	txContract := NewTxContract(tx)

	prevUTXOS := []*utxo.UTXO{
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(100), PubKeyHash: contractAcc.GetPubKeyHash(), Contract: "0123456789abcdef"},
			Txid:     []byte{0x88},
			TxIndex:  0,
			UtxoType: utxo.UtxoNormal,
		},
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(1010), PubKeyHash: contractAcc.GetPubKeyHash(), Contract: "0123456789abcdef"},
			Txid:     []byte{0x88},
			TxIndex:  1,
			UtxoType: utxo.UtxoNormal,
		},
	}

	total, err := txContract.GetTotalBalance(prevUTXOS)
	assert.Equal(t, common.NewAmount(1094), total)
	assert.Nil(t, err)

	tx.Vout[0].Value = common.NewAmount(9999)
	total, err = txContract.GetTotalBalance(prevUTXOS)
	assert.Nil(t, total)
	assert.Equal(t, transaction.ErrInsufficientBalance, err)
}

func TestTransaction_GetContractAddress(t *testing.T) {

	tests := []struct {
		name        string
		addr        string
		expectedRes string
	}{
		{
			name:        "ContainsContractAddress",
			addr:        "cavQdWxvUQU1HhBg1d7zJFwhf31SUaQwop",
			expectedRes: "cavQdWxvUQU1HhBg1d7zJFwhf31SUaQwop",
		},
		{
			name:        "ContainsUserAddress",
			addr:        "dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf",
			expectedRes: "",
		},
		{
			name:        "EmptyInput",
			addr:        "",
			expectedRes: "",
		},
		{
			name:        "InvalidAddress",
			addr:        "dsdGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf",
			expectedRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := account.NewTransactionAccountByAddress(account.NewAddress(tt.addr))
			tx := &transaction.Transaction{
				nil,
				nil,
				[]transactionbase.TXOutput{
					{nil,
						acc.GetPubKeyHash(),
						"",
					},
				},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				transaction.TxTypeContract,
			}
			ctx := NewTxContract(tx)
			if ctx != nil {
				assert.Equal(t, account.NewAddress(tt.expectedRes), ctx.GetContractAddress())
			}
		})
	}
}

func TestNewGasRewardTx(t *testing.T) {
	tests := []struct {
		name           string
		actualGasCount *common.Amount
		gasPrice       *common.Amount
		expectedResult transaction.Transaction
		expectedOk     bool
	}{
		{
			name:           "zero actualGasCount",
			actualGasCount: common.NewAmount(0),
			gasPrice:       common.NewAmount(2),
			expectedResult: transaction.Transaction{},
			expectedOk:     false,
		},
		{
			name:           "zero gasPrice",
			actualGasCount: common.NewAmount(30),
			gasPrice:       common.NewAmount(0),
			expectedResult: transaction.Transaction{},
			expectedOk:     false,
		},
		{
			name:           "successful",
			actualGasCount: common.NewAmount(30),
			gasPrice:       common.NewAmount(2),
			expectedResult: transaction.Transaction{
				ID: []byte{0xee, 0x6f, 0x24, 0x67, 0xf4, 0xcb, 0xff, 0xed, 0xe5, 0x6a, 0x16, 0xbc, 0xe3, 0xa4, 0xd9, 0x72, 0x12, 0xeb, 0x32, 0xb4, 0xf0, 0x2b, 0x52, 0x2f, 0xc4, 0xdc, 0x44, 0x2, 0x95, 0x6d, 0x76, 0x78},
				Vin: []transactionbase.TXInput{
					{
						Txid:      []uint8(nil),
						Vout:      -1,
						Signature: []uint8{0x0, 0x7b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3},
						PubKey:    []uint8{0x4d, 0x69, 0x6e, 0x65, 0x72, 0x20, 0x47, 0x61, 0x73, 0x20, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64, 0x73},
					},
				},
				Vout: []transactionbase.TXOutput{
					{
						Value:      common.NewAmount(60),
						PubKeyHash: account.PubKeyHash{0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80, 0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49, 0xe5, 0x27, 0xf0, 0x42, 0x5d},
						Contract:   "",
					},
				},
				Tip:        common.NewAmount(0),
				GasLimit:   common.NewAmount(0),
				GasPrice:   common.NewAmount(0),
				CreateTime: 0,
				Type:       transaction.TxTypeGasReward,
			},
			expectedOk: true,
		},
	}

	to := account.NewTransactionAccountByAddress(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := NewGasRewardTx(to, 3, tt.actualGasCount, tt.gasPrice, 123)
			assert.Equal(t, tt.expectedOk, ok)
			if tt.expectedOk {
				assert.Equal(t, tt.expectedResult.ID, result.ID)
				assert.Equal(t, tt.expectedResult.Vin, result.Vin)
				assert.Equal(t, tt.expectedResult.Vout, result.Vout)
				assert.Equal(t, common.NewAmount(0), result.Tip)
				assert.Equal(t, common.NewAmount(0), result.GasLimit)
				assert.Equal(t, common.NewAmount(0), result.GasPrice)
				assert.Equal(t, transaction.TxTypeGasReward, result.Type)
				assert.Equal(t, tt.expectedResult.Hash(), result.Hash())
			} else {
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestNewCoinbaseTX(t *testing.T) {
	t1 := NewCoinbaseTX(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 0, common.NewAmount(0))
	t2 := NewCoinbaseTX(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 0, common.NewAmount(0))
	expectVin := transactionbase.TXInput{nil, -1, []byte{0, 0, 0, 0, 0, 0, 0, 0}, []byte("Reward to 'dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB'")}
	expectVout := transactionbase.TXOutput{transaction.Subsidy, account.PubKeyHash([]byte{0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80, 0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49, 0xe5, 0x27, 0xf0, 0x42, 0x5d}), ""}
	assert.Equal(t, 1, len(t1.Vin))
	assert.Equal(t, expectVin, t1.Vin[0])
	assert.Equal(t, 1, len(t1.Vout))
	assert.Equal(t, expectVout, t1.Vout[0])
	assert.Equal(t, common.NewAmount(0), t1.Tip)

	// Assert that ltransaction.NewCoinbaseTX is deterministic (i.e. >1 coinbaseTXs in a block would have identical txid)
	assert.Equal(t, t1, t2)

	t3 := NewCoinbaseTX(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 1, common.NewAmount(0))

	assert.NotEqual(t, t1, t3)
	assert.NotEqual(t, t1.ID, t3.ID)
}

func TestNewUTXOTransaction(t *testing.T) {
	from := account.NewAccountByPrivateKey("bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71")
	to := account.NewAccountByPrivateKey("bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72")
	utxos := []*utxo.UTXO{
		{
			TXOutput: transactionbase.TXOutput{
				Value:      common.NewAmount(9001),
				PubKeyHash: from.GetPubKeyHash(),
				Contract:   "contract",
			},
			Txid:     []byte{0x88},
			TxIndex:  0,
			UtxoType: utxo.UtxoNormal,
		},
	}
	params := transaction.NewSendTxParam(
		from.GetAddress(),
		from.GetKeyPair(),
		to.GetAddress(),
		common.NewAmount(10),
		common.NewAmount(1),
		common.NewAmount(3000),
		common.NewAmount(2),
		"contract")

	expected := transaction.Transaction{
		ID: []uint8{0x58, 0xff, 0xdb, 0x5, 0xd1, 0xde, 0x67, 0xba, 0xb, 0xe9, 0x54, 0x6b, 0xad, 0xa0, 0xf1, 0x49, 0xde, 0xd2, 0xba, 0xf, 0x67, 0x6a, 0xb8, 0x31, 0xd6, 0x78, 0xe5, 0xc1, 0x72, 0xcb, 0x47, 0x41},
		Vin: []transactionbase.TXInput{
			{
				Txid:      []uint8{0x88},
				Vout:      0,
				Signature: []uint8{0xff, 0xd7, 0x6e, 0x63, 0x5b, 0x60, 0xc9, 0x71, 0xde, 0xc5, 0xa8, 0xd2, 0x55, 0x5d, 0x3f, 0x73, 0x40, 0x43, 0xc0, 0x4a, 0x9f, 0x5c, 0x24, 0x31, 0x68, 0xfd, 0xfe, 0x71, 0xcc, 0xfd, 0x1d, 0xc9, 0x3a, 0x92, 0x3e, 0xd6, 0x11, 0xaf, 0xfe, 0x72, 0xf0, 0x64, 0x9d, 0x6d, 0xd2, 0xba, 0x94, 0xf7, 0xa, 0xd5, 0xff, 0x2b, 0x8c, 0x83, 0xc9, 0x44, 0xd9, 0x91, 0xd4, 0xe2, 0x14, 0x36, 0x6d, 0x92, 0x1},
				PubKey:    []uint8{0x32, 0x4f, 0xe9, 0x1c, 0x75, 0x0, 0x8a, 0x1c, 0xe4, 0x21, 0x6d, 0xd7, 0x38, 0x73, 0xeb, 0xf9, 0x9b, 0xd1, 0xa1, 0x48, 0x3b, 0xe, 0x32, 0xeb, 0xd5, 0x50, 0x4, 0x86, 0x75, 0x9c, 0x84, 0x99, 0xe4, 0xbd, 0xfa, 0x79, 0xbd, 0x63, 0xf, 0x46, 0xed, 0x57, 0x85, 0x38, 0xc7, 0x4, 0xf4, 0x2a, 0xd9, 0x65, 0x64, 0xc0, 0xa7, 0xf7, 0xd9, 0x57, 0xb3, 0xed, 0x84, 0xf5, 0xee, 0xad, 0x4d, 0x71},
			},
		},
		Vout: []transactionbase.TXOutput{
			{
				Value:      common.NewAmount(0),
				PubKeyHash: account.PubKeyHash{0x5a, 0x62, 0xb7, 0xc, 0x40, 0x81, 0xbf, 0xd2, 0x96, 0xb6, 0x42, 0xc0, 0x33, 0x93, 0xcc, 0x57, 0x61, 0xce, 0xeb, 0xb2, 0xac},
				Contract:   "contract",
			},
			{
				Value:      common.NewAmount(10),
				PubKeyHash: account.PubKeyHash{0x5a, 0x62, 0xb7, 0xc, 0x40, 0x81, 0xbf, 0xd2, 0x96, 0xb6, 0x42, 0xc0, 0x33, 0x93, 0xcc, 0x57, 0x61, 0xce, 0xeb, 0xb2, 0xac},
				Contract:   "",
			},
			{
				Value:      common.NewAmount(2990),
				PubKeyHash: account.PubKeyHash{0x5a, 0xf8, 0xbf, 0x23, 0x39, 0x70, 0xf0, 0x9b, 0x65, 0x31, 0x98, 0xca, 0xed, 0x6c, 0xb6, 0x13, 0xb, 0x77, 0xd, 0x6f, 0x5},
				Contract:   "",
			},
		},
		Tip:        common.NewAmount(1),
		GasLimit:   common.NewAmount(3000),
		GasPrice:   common.NewAmount(2),
		CreateTime: 0,
		Type:       transaction.TxTypeNormal,
	}
	result, err := NewUTXOTransaction(transaction.TxTypeNormal, utxos, params)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Vin, result.Vin)
	assert.Equal(t, expected.Vout, result.Vout)
	assert.Equal(t, expected.Tip, result.Tip)
	assert.Equal(t, expected.GasLimit, result.GasLimit)
	assert.Equal(t, expected.GasPrice, result.GasPrice)
	assert.Equal(t, expected.Type, result.Type)
	assert.Nil(t, err)
}

func TestPrepareInputLists(t *testing.T) {
	utxos := []*utxo.UTXO{
		{
			TXOutput:    transactionbase.TXOutput{},
			Txid:        []byte{0x88},
			TxIndex:     0,
			UtxoType:    utxo.UtxoNormal,
			PrevUtxoKey: nil,
			NextUtxoKey: nil,
		},
		{
			TXOutput:    transactionbase.TXOutput{},
			Txid:        []byte{0x89},
			TxIndex:     1,
			UtxoType:    utxo.UtxoNormal,
			PrevUtxoKey: nil,
			NextUtxoKey: nil,
		},
	}
	expected := []transactionbase.TXInput{
		{Txid: []byte{0x88}, Vout: 0, Signature: []byte("signature"), PubKey: []byte("pubkey")},
		{Txid: []byte{0x89}, Vout: 1, Signature: []byte("signature"), PubKey: []byte("pubkey")},
	}

	result := prepareInputLists(utxos, []byte("pubkey"), []byte("signature"))
	assert.Equal(t, expected, result)
}

func TestPrepareOutputLists(t *testing.T) {
	from := account.NewTransactionAccountByAddress(account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"))
	to := account.NewTransactionAccountByAddress(account.NewAddress("cavQdWxvUQU1HhBg1d7zJFwhf31SUaQwop"))

	expected := []transactionbase.TXOutput{
		{
			Value:      common.NewAmount(0),
			PubKeyHash: to.GetPubKeyHash(),
			Contract:   "0123456789abcdef",
		},
		{
			Value:      common.NewAmount(1000),
			PubKeyHash: to.GetPubKeyHash(),
			Contract:   "",
		},
		{
			Value:      common.NewAmount(20),
			PubKeyHash: from.GetPubKeyHash(),
			Contract:   "",
		},
	}

	result := prepareOutputLists(from, to, common.NewAmount(1000), common.NewAmount(20), "0123456789abcdef")
	assert.Equal(t, expected, result)
}

func TestDescribeTransaction(t *testing.T) {
	acc := account.NewAccount()
	normalTx := &transaction.Transaction{
		ID: []byte{0x88},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0x88}, Vout: 0, Signature: nil, PubKey: []byte{0}},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
		},
		Tip:        common.NewAmount(1),
		GasLimit:   common.NewAmount(3000),
		GasPrice:   common.NewAmount(2),
		CreateTime: 0,
		Type:       transaction.TxTypeNormal,
	}

	db := storage.NewRamStorage()
	defer db.Close()
	cache := utxo.NewUTXOCache(db)
	utxoIndex := lutxo.NewUTXOIndex(cache)

	// invalid normalTx.Vin[0] PubKey
	sender, recipient, amount, tip, err := DescribeTransaction(utxoIndex, normalTx)
	assert.Nil(t, sender)
	assert.Nil(t, recipient)
	assert.Nil(t, amount)
	assert.Nil(t, tip)
	assert.Equal(t, errors.New("public key not correct"), err)
	normalTx.Vin[0].PubKey = acc.GetKeyPair().GetPublicKey()

	// utxo not in cache
	sender, recipient, amount, tip, err = DescribeTransaction(utxoIndex, normalTx)
	assert.Nil(t, sender)
	assert.Nil(t, recipient)
	assert.Nil(t, amount)
	assert.Nil(t, tip)
	assert.Equal(t, errors.New("key is invalid"), err)
	utxoIndex.AddUTXO(normalTx.Vout[0], normalTx.ID, 0)

	// successful normal tx
	sender, recipient, amount, tip, err = DescribeTransaction(utxoIndex, normalTx)
	assert.Equal(t, acc.GetAddress(), *sender)
	assert.Equal(t, acc.GetAddress(), *recipient)
	assert.Equal(t, common.NewAmount(10), amount)
	assert.Equal(t, common.NewAmount(10), tip)
	assert.Nil(t, err)

	// make contractSendTx
	contractAcc := account.NewContractTransactionAccount()
	contractTx := &transaction.Transaction{
		ID: []byte{0x89},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0x89}, Vout: 0, Signature: nil, PubKey: contractAcc.GetPubKeyHash()},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(3), PubKeyHash: contractAcc.GetPubKeyHash(), Contract: "contract1"},
		},
		Tip:        common.NewAmount(1),
		GasLimit:   common.NewAmount(3000),
		GasPrice:   common.NewAmount(2),
		CreateTime: 0,
		Type:       transaction.TxTypeContractSend,
	}

	txo := transactionbase.TXOutput{common.NewAmount(5), contractAcc.GetPubKeyHash(), ""}
	utxoIndex.AddUTXO(txo, contractTx.ID, 0)
	sender, recipient, amount, tip, err = DescribeTransaction(utxoIndex, contractTx)
	assert.Equal(t, contractAcc.GetAddress(), *sender)
	assert.Equal(t, &account.Address{}, recipient)
	assert.Equal(t, common.NewAmount(0), amount)
	assert.Equal(t, common.NewAmount(2), tip)
	assert.Nil(t, err)
}
