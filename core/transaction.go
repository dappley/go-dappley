package core

import (
	"bytes"
	"crypto/ecdsa"
	//	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"math/big"

	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/storage"
	"github.com/gogo/protobuf/proto"
)

const subsidy = 10

var (
	ErrInsufficientFund = errors.New("ERROR: The balance is insufficient")
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
	Tip  int64
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// Serialize returns a serialized Transaction
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// Hash returns the hash of the Transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// Sign signs each input of a Transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		privData, err := secp256k1.FromECDSAPrivateKey(&privKey)
		if err != nil {
			return
		}

		signature, error := secp256k1.Sign(txCopy.ID, privData)
		if error != nil {
			return
		}

		tx.Vin[inID].Signature = signature

	}
}

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip}

	return txCopy
}

// Verify verifies signatures of Transaction inputs
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {

	var verifyResult bool
	var error1 error

	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	//	curve := elliptic.P256()
	curve := secp256k1.S256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		originPub, err := secp256k1.FromECDSAPublicKey(&rawPubKey)
		if err != nil {
			return false
		}

		verifyResult, error1 = secp256k1.Verify(txCopy.ID, vin.Signature, originPub)

		if error1 != nil || verifyResult == false {
			return false
		}
	}

	return true
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to, data string) Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}

	randData := make([]byte, 20)
	_, err := rand.Read(randData)
	if err != nil {
		log.Panic(err)
	}
	data = fmt.Sprintf("%s - %x", data, randData)
	txin := TXInput{nil, -1, nil, []byte(data)}
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}, 0}
	tx.ID = tx.Hash()

	return tx
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(db storage.Storage, from, to Address, amount int, keypair KeyPair, bc *Blockchain, tip int64) (Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput
	var validOutputs []TXOutputStored

	pubKeyHash := HashPubKey(keypair.PublicKey)
	sum := 0

	if len(GetAddressUTXOs(pubKeyHash, db)) < 1 {
		return Transaction{}, ErrInsufficientFund
	}
	for _, v := range GetAddressUTXOs(pubKeyHash, db) {
		sum += v.Value
		validOutputs = append(validOutputs, v)
		if sum >= amount {
			break
		}
	}

	if sum < amount {
		return Transaction{}, ErrInsufficientFund
	}

	// Build a list of inputs
	for _, out := range validOutputs {
		input := TXInput{out.Txid, out.TxIndex, nil, keypair.PublicKey}
		inputs = append(inputs, input)

	}
	// Build a list of outputs
	outputs = append(outputs, *NewTXOutput(amount, to.Address))
	if sum > amount {
		outputs = append(outputs, *NewTXOutput(sum-amount, from.Address)) // a change
	}

	tx := Transaction{nil, inputs, outputs, tip}
	tx.ID = tx.Hash()
	bc.SignTransaction(&tx, keypair.PrivateKey)

	return tx, nil
}

//for add balance
func NewUTXOTransactionforAddBalance(to Address, amount int, keypair KeyPair, bc *Blockchain, tip int64) (Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput

	// Build a list of outputs
	outputs = append(outputs, *NewTXOutput(amount, to.Address))

	tx := Transaction{nil, inputs, outputs, tip}
	tx.ID = tx.Hash()
	bc.SignTransaction(&tx, keypair.PrivateKey)

	return tx, nil
}

// String returns a human-readable representation of a transaction
func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("\n--- Transaction %x:", tx.ID))

	for i, input := range tx.Vin {

		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}
	lines = append(lines, "\n")

	return strings.Join(lines, "\n")
}

func (tx *Transaction) ToProto() proto.Message {

	vinArray := []*corepb.TXInput{}
	for _, txin := range tx.Vin {
		vinArray = append(vinArray, txin.ToProto().(*corepb.TXInput))
	}

	voutArray := []*corepb.TXOutput{}
	for _, txout := range tx.Vout {
		voutArray = append(voutArray, txout.ToProto().(*corepb.TXOutput))
	}

	return &corepb.Transaction{
		ID:   tx.ID,
		Vin:  vinArray,
		Vout: voutArray,
		Tip:  tx.Tip,
	}
}

func (tx *Transaction) FromProto(pb proto.Message) {
	tx.ID = pb.(*corepb.Transaction).ID
	tx.Tip = pb.(*corepb.Transaction).Tip

	vinArray := []TXInput{}
	txin := TXInput{}
	for _, txinpb := range pb.(*corepb.Transaction).Vin {
		txin.FromProto(txinpb)
		vinArray = append(vinArray, txin)
	}
	tx.Vin = vinArray

	voutArray := []TXOutput{}
	txout := TXOutput{}
	for _, txoutpb := range pb.(*corepb.Transaction).Vout {
		txout.FromProto(txoutpb)
		voutArray = append(voutArray, txout)
	}
	tx.Vout = voutArray
}
