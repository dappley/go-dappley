// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/util"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func GenerateFakeTxInputs() []TXInput {
	return []TXInput{
		{getAoB(2), 10, getAoB(2), getAoB(2)},
		{getAoB(2), 5, getAoB(2), getAoB(2)},
	}
}

func GenerateFakeTxOutputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(1), PubKeyHash{getAoB(2)}, ""},
		{common.NewAmount(2), PubKeyHash{getAoB(2)}, ""},
	}
}

func TestTrimmedCopy(t *testing.T) {
	var t1 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  2,
	}

	t2 := t1.TrimmedCopy()

	t3 := NewCoinbaseTX(NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F"), "", 0, common.NewAmount(0))
	t4 := t3.TrimmedCopy()
	assert.Equal(t, t1.ID, t2.ID)
	assert.Equal(t, t1.Tip, t2.Tip)
	assert.Equal(t, t1.Vout, t2.Vout)
	for index, vin := range t2.Vin {
		assert.Nil(t, vin.Signature)
		assert.Nil(t, vin.PubKey)
		assert.Equal(t, t1.Vin[index].Txid, vin.Txid)
		assert.Equal(t, t1.Vin[index].Vout, vin.Vout)
	}

	assert.Equal(t, t3.ID, t4.ID)
	assert.Equal(t, t3.Tip, t4.Tip)
	assert.Equal(t, t3.Vout, t4.Vout)
	for index, vin := range t4.Vin {
		assert.Nil(t, vin.Signature)
		assert.Nil(t, vin.PubKey)
		assert.Equal(t, t3.Vin[index].Txid, vin.Txid)
		assert.Equal(t, t3.Vin[index].Vout, vin.Vout)
	}
}

func TestSign(t *testing.T) {
	// Fake a key pair
	privKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("fakefakefakefakefakefakefakefakefakefake")))
	ecdsaPubKey, _ := secp256k1.FromECDSAPublicKey(&privKey.PublicKey)
	pubKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	pubKeyHash, _ := NewUserPubKeyHash(pubKey)

	// Previous transactions containing UTXO of the Address
	prevTXs := []*UTXO{
		{TXOutput{common.NewAmount(13), pubKeyHash, ""}, []byte("01"), 0},
		{TXOutput{common.NewAmount(13), pubKeyHash, ""}, []byte("02"), 0},
		{TXOutput{common.NewAmount(13), pubKeyHash, ""}, []byte("03"), 0},
	}

	// New transaction to be signed (paid from the fake account)
	txin := []TXInput{
		{[]byte{1}, 0, nil, pubKey},
		{[]byte{3}, 0, nil, pubKey},
		{[]byte{3}, 2, nil, pubKey},
	}
	txout := []TXOutput{
		{common.NewAmount(19), pubKeyHash, ""},
	}
	tx := Transaction{nil, txin, txout, 0}

	// Sign the transaction
	err := tx.Sign(*privKey, prevTXs)
	if assert.Nil(t, err) {
		// Assert that the signatures were created by the fake key pair
		for i, vin := range tx.Vin {

			if assert.NotNil(t, vin.Signature) {
				txCopy := tx.TrimmedCopy()
				txCopy.Vin[i].Signature = nil
				txCopy.Vin[i].PubKey = pubKeyHash.GetPubKeyHash()

				verified, err := secp256k1.Verify(txCopy.Hash(), vin.Signature, ecdsaPubKey)
				assert.Nil(t, err)
				assert.True(t, verified)
			}
		}
	}
}

func TestVerifyCoinbaseTransaction(t *testing.T) {
	var prevTXs = map[string]Transaction{}

	var t1 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  2,
	}

	var t2 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  5,
	}
	var t3 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  10,
	}
	var t4 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  20,
	}
	prevTXs[string(t1.ID)] = t2
	prevTXs[string(t2.ID)] = t3
	prevTXs[string(t3.ID)] = t4

	// test verifying coinbase transactions
	var t5 = NewCoinbaseTX(NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F"), "", 5, common.NewAmount(0))
	bh1 := make([]byte, 8)
	binary.BigEndian.PutUint64(bh1, 5)
	txin1 := TXInput{nil, -1, bh1, []byte(nil)}
	txout1 := NewTXOutput(common.NewAmount(10), NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F"))
	var t6 = Transaction{nil, []TXInput{txin1}, []TXOutput{*txout1}, 0}

	// test valid coinbase transaction
	assert.True(t, t5.Verify(&UTXOIndex{}, 5))
	assert.True(t, t6.Verify(&UTXOIndex{}, 5))

	// test coinbase transaction with incorrect blockHeight
	assert.False(t, t5.Verify(&UTXOIndex{}, 10))

	// test coinbase transaction with incorrect subsidy
	bh2 := make([]byte, 8)
	binary.BigEndian.PutUint64(bh2, 5)
	txin2 := TXInput{nil, -1, bh2, []byte(nil)}
	txout2 := NewTXOutput(common.NewAmount(20), NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F"))
	var t7 = Transaction{nil, []TXInput{txin2}, []TXOutput{*txout2}, 0}
	assert.False(t, t7.Verify(&UTXOIndex{}, 5))

}

func TestVerifyNoCoinbaseTransaction(t *testing.T) {
	// Fake a key pair
	privKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("fakefakefakefakefakefakefakefakefakefake")))
	privKeyByte, _ := secp256k1.FromECDSAPrivateKey(privKey)
	pubKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	pubKeyHash, _ := NewUserPubKeyHash(pubKey)
	//Address := KeyPair{*privKey, pubKey}.GenerateAddress()

	// Fake a wrong key pair
	wrongPrivKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("FAKEfakefakefakefakefakefakefakefakefake")))
	wrongPrivKeyByte, _ := secp256k1.FromECDSAPrivateKey(wrongPrivKey)
	wrongPubKey := append(wrongPrivKey.PublicKey.X.Bytes(), wrongPrivKey.PublicKey.Y.Bytes()...)
	//wrongPubKeyHash, _ := NewUserPubKeyHash(wrongPubKey)
	//wrongAddress := KeyPair{*wrongPrivKey, wrongPubKey}.GenerateAddress()
	utxoIndex := NewUTXOIndex()
	utxoIndex.index = map[string][]*UTXO{
		string(pubKeyHash.GetPubKeyHash()): {
			{TXOutput{common.NewAmount(4), pubKeyHash, ""}, []byte{1}, 0},
			{TXOutput{common.NewAmount(3), pubKeyHash, ""}, []byte{2}, 1},
		},
	}

	// Prepare a transaction to be verified
	txin := []TXInput{{[]byte{1}, 0, nil, pubKey}}
	txin1 := append(txin, TXInput{[]byte{2}, 1, nil, pubKey})      // Normal test
	txin2 := append(txin, TXInput{[]byte{2}, 1, nil, wrongPubKey}) // previous not found with wrong pubkey
	txin3 := append(txin, TXInput{[]byte{3}, 1, nil, pubKey})      // previous not found with wrong Txid
	txin4 := append(txin, TXInput{[]byte{2}, 2, nil, pubKey})      // previous not found with wrong TxIndex
	pbh, _ := NewUserPubKeyHash(pubKey)
	txout := []TXOutput{{common.NewAmount(7), pbh, ""}}
	txout2 := []TXOutput{{common.NewAmount(8), pbh, ""}} //Vout amount > Vin amount

	tests := []struct {
		name     string
		tx       Transaction
		signWith []byte
		ok       bool
	}{
		{"normal", Transaction{nil, txin1, txout, 0}, privKeyByte, true},
		{"previous tx not found with wrong pubkey", Transaction{nil, txin2, txout, 0}, privKeyByte, false},
		{"previous tx not found with wrong Txid", Transaction{nil, txin3, txout, 0}, privKeyByte, false},
		{"previous tx not found with wrong TxIndex", Transaction{nil, txin4, txout, 0}, privKeyByte, false},
		{"Amount invalid", Transaction{nil, txin1, txout2, 0}, privKeyByte, false},
		{"Sign invalid", Transaction{nil, txin1, txout, 0}, wrongPrivKeyByte, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate signatures for all tx inputs
			for i := range tt.tx.Vin {
				txCopy := tt.tx.TrimmedCopy()
				txCopy.Vin[i].Signature = nil
				txCopy.Vin[i].PubKey = pubKeyHash.GetPubKeyHash()
				signature, _ := secp256k1.Sign(txCopy.Hash(), tt.signWith)
				tt.tx.Vin[i].Signature = signature
			}

			// Verify the signatures
			result := tt.tx.Verify(utxoIndex, 0)
			assert.Equal(t, tt.ok, result)
		})
	}
}

func TestNewCoinbaseTX(t *testing.T) {
	t1 := NewCoinbaseTX(NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 0, common.NewAmount(0))
	expectVin := TXInput{nil, -1, []byte{0, 0, 0, 0, 0, 0, 0, 0}, []byte("Reward to 'dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB'")}
	expectVout := TXOutput{common.NewAmount(10), PubKeyHash{[]byte{0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80, 0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49, 0xe5, 0x27, 0xf0, 0x42, 0x5d}}, ""}
	assert.Equal(t, 1, len(t1.Vin))
	assert.Equal(t, expectVin, t1.Vin[0])
	assert.Equal(t, 1, len(t1.Vout))
	assert.Equal(t, expectVout, t1.Vout[0])
	assert.Equal(t, uint64(0), t1.Tip)

	t2 := NewCoinbaseTX(NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 0, common.NewAmount(0))

	// Assert that NewCoinbaseTX is deterministic (i.e. >1 coinbaseTXs in a block would have identical txid)
	assert.Equal(t, t1, t2)

	t3 := NewCoinbaseTX(NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 1, common.NewAmount(0))

	assert.NotEqual(t, t1, t3)
	assert.NotEqual(t, t1.ID, t3.ID)
}

//test IsCoinBase function
func TestIsCoinBase(t *testing.T) {
	var t1 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  2,
	}

	assert.False(t, t1.IsCoinbase())

	t2 := NewCoinbaseTX(NewAddress("13ZRUc4Ho3oK3Cw56PhE5rmaum9VBeAn5F"), "", 0, common.NewAmount(0))

	assert.True(t, t2.IsCoinbase())

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

func TestTransaction_IsRewardTx(t *testing.T) {
	tests := []struct {
		name        string
		tx          Transaction
		expectedRes bool
	}{
		{"normal", NewRewardTx(1, map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "9"}), true},
		{"no rewards", NewRewardTx(1, nil), true},
		{"coinbase", NewCoinbaseTX(NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"), "", 5, common.NewAmount(0)), false},
		{"normal tx", *MockTransaction(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.tx.IsRewardTx())
		})
	}
}

func TestTransaction_Proto(t *testing.T) {
	t1 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  5,
	}

	pb := t1.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.Transaction{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	t2 := Transaction{}
	t2.FromProto(newpb)

	assert.Equal(t, t1, t2)
}

func TestTransaction_FindTxInUtxoPool(t *testing.T) {
	//prepare utxo pool
	pubkey := []byte("12345678901234567890123456789012")
	pubkeyHash, _ := NewUserPubKeyHash(pubkey)

	Txin := MockTxInputsWithPubkey(pubkey)
	Txin2 := MockTxInputsWithPubkey(pubkey)
	utxo1 := &UTXO{TXOutput{common.NewAmount(10), pubkeyHash, ""}, Txin[0].Txid, Txin[0].Vout}
	utxo2 := &UTXO{TXOutput{common.NewAmount(9), pubkeyHash, ""}, Txin[1].Txid, Txin[1].Vout}
	utxo3 := &UTXO{TXOutput{common.NewAmount(9), pubkeyHash, ""}, Txin2[0].Txid, Txin2[0].Vout}
	utxo4 := &UTXO{TXOutput{common.NewAmount(9), pubkeyHash, ""}, Txin2[1].Txid, Txin2[1].Vout}
	utxoPool := NewUTXOIndex()
	utxoPool.index[string(pubkeyHash.GetPubKeyHash())] = []*UTXO{utxo1, utxo2, utxo3, utxo4}

	tx := MockTransaction()
	txins, _ := tx.FindAllTxinsInUtxoPool(*utxoPool)
	assert.Nil(t, txins)
	tx.Vin = Txin
	txins, _ = tx.FindAllTxinsInUtxoPool(*utxoPool)
	assert.NotNil(t, txins)
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
			addr := NewAddress(tt.addr)
			pkh, _ := addr.GetPubKeyHash()
			tx := Transaction{
				nil,
				nil,
				[]TXOutput{
					{nil,
						PubKeyHash{pkh},
						"",
					},
				},
				0,
			}

			assert.Equal(t, NewAddress(tt.expectedRes), tx.GetContractAddress())
		})
	}
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
			sc := new(ScEngineMock)
			contract := "helloworld!"
			if tt.expectContractRun {
				sc.On("ImportSourceCode", contract)
				sc.On("Execute")
			}

			scUtxo := UTXO{
				TxIndex: 0,
				Txid:    nil,
				TXOutput: TXOutput{
					PubKeyHash: PubKeyHash{[]byte(tt.scAddr)},
					Contract:   contract,
				},
			}

			tx := Transaction{
				Vout: []TXOutput{{nil, PubKeyHash{[]byte(tt.toAddr)}, ""}},
			}

			index := NewUTXOIndex()
			if tt.scAddr != "" {
				index.addUTXO(scUtxo.TXOutput, nil, 0)
			}

			tx.Execute(*index, nil, nil, sc)
		})
	}
}

func TestTransaction_MatchRewards(t *testing.T) {

	tests := []struct {
		name          string
		tx            *Transaction
		rewardStorage map[string]string
		expectedRes   bool
	}{
		{"normal",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{{
					common.NewAmount(1),
					PubKeyHash{[]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}},
					"",
				}},
				0,
			},
			map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1"},
			true,
		},
		{"emptyVout",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{},
				0,
			},
			map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1"},
			false,
		},
		{"emptyRewardMap",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{{
					common.NewAmount(1),
					PubKeyHash{[]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}},
					"",
				}},
				0,
			},
			nil,
			false,
		},
		{"Wrong address",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{{
					common.NewAmount(1),
					PubKeyHash{[]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}},
					"",
				}},
				0,
			},
			map[string]string{"dXnq2R6SzRNUt7ZsNAqyZc2P9ziF6vYekB": "1"},
			false,
		},
		{"Wrong amount",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{{
					common.NewAmount(3),
					PubKeyHash{[]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}},
					"",
				}},
				0,
			},
			map[string]string{"dXnq2R6SzRNUt7ZsNAqyZc2P9ziF6vYekB": "1"},
			false,
		},
		{"twoAddresses",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{{
					common.NewAmount(1),
					PubKeyHash{[]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}},
					"",
				},
					{
						common.NewAmount(4),
						PubKeyHash{[]byte{
							90, 13, 39, 130, 118, 11, 160, 130, 83, 126, 86, 102, 252, 178, 87,
							218, 57, 174, 123, 244, 229}},
						"",
					}},
				0,
			},
			map[string]string{
				"dEcqjSgREFi9gTCbAWpEQ3kbPxgsBzzhWS": "4",
				"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1",
			},
			true,
		},
		{"MoreRewards",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{{
					common.NewAmount(1),
					PubKeyHash{[]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}},
					"",
				},
					{
						common.NewAmount(4),
						PubKeyHash{[]byte{
							90, 13, 39, 130, 118, 11, 160, 130, 83, 126, 86, 102, 252, 178, 87,
							218, 57, 174, 123, 244, 229}},
						"",
					}},
				0,
			},
			map[string]string{
				"dEcqjSgREFi9gTCbAWpEQ3kbPxgsBzzhWS": "4",
				"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1",
				"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6fYekB": "3",
			},
			false,
		},
		{"MoreVout",
			&Transaction{
				nil,
				[]TXInput{{nil, -1, nil, rewardTxData}},
				[]TXOutput{{
					common.NewAmount(1),
					PubKeyHash{[]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}},
					"",
				},
					{
						common.NewAmount(4),
						PubKeyHash{[]byte{
							90, 13, 39, 130, 118, 11, 160, 130, 83, 126, 86, 102, 252, 178, 87,
							218, 57, 174, 123, 244, 229}},
						"",
					}},
				0,
			},
			map[string]string{
				"dEcqjSgREFi9gTCbAWpEQ3kbPxgsBzzhWS": "4",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.tx.MatchRewards(tt.rewardStorage))
		})
	}

}
