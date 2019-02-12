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
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/crypto/byteutils"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

var subsidy = common.NewAmount(10000000)

const ContractTxouputIndex = 0

var rewardTxData = []byte("Distribute X Rewards")

var (
	ErrInsufficientFund  = errors.New("transaction: insufficient balance")
	ErrInvalidAmount     = errors.New("transaction: invalid amount (must be > 0)")
	ErrTXInputNotFound   = errors.New("transaction: transaction input not found")
	ErrNewUserPubKeyHash = errors.New("transaction: create pubkeyhash error")
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
	Tip  *common.Amount
}

type TxIndex struct {
	BlockId    []byte
	BlockIndex int
}

func (tx *Transaction) IsCoinbase() bool {

	if !tx.isVinCoinbase() {
		return false
	}

	if len(tx.Vout) != 1 {
		return false
	}

	if bytes.Equal(tx.Vin[0].PubKey, rewardTxData) {
		return false
	}

	return true
}

func (tx *Transaction) IsRewardTx() bool {

	if !tx.isVinCoinbase() {
		return false
	}

	if !bytes.Equal(tx.Vin[0].PubKey, rewardTxData) {
		return false
	}

	return true
}

// IsContract returns true if tx deploys/executes a smart contract; false otherwise
func (tx *Transaction) IsContract() bool {
	if len(tx.Vout) == 0 {
		return false
	}
	isContract, _ := tx.Vout[ContractTxouputIndex].PubKeyHash.IsContract()
	return isContract
}

func (tx *Transaction) IsExecutionContract() bool {
	if tx.IsContract() && !strings.Contains(tx.GetContract(), scheduleFuncName) {
		return true
	}
	return false
}

// IsFromContract returns true if tx is generated from a contract execution; false otherwise
func (tx *Transaction) IsFromContract() bool {
	if len(tx.Vin) == 0 {
		return false
	}
	for _, vin := range tx.Vin {
		pubKey := PubKeyHash(vin.PubKey)
		if IsContract, _ := pubKey.IsContract(); !IsContract {
			return false
		}
	}
	return true
}

func (tx *Transaction) isVinCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// Serialize returns a serialized Transaction
func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		logger.Panic(err)
	}

	return encoded.Bytes()
}

// Describe reverse-engineers the high-level description of a transaction
func (tx *Transaction) Describe(index UTXOIndex) (sender, recipient *Address, amount, tip *common.Amount, error error) {
	var receiverAddress Address
	vinPubKey := tx.Vin[0].PubKey
	pubKeyHash := PubKeyHash([]byte(""))
	inputAmount := common.NewAmount(0)
	outputAmount := common.NewAmount(0)
	payoutAmount := common.NewAmount(0)
	for _, vin := range tx.Vin {
		if bytes.Compare(vin.PubKey, vinPubKey) == 0 {
			switch {
			case tx.IsRewardTx():
				pubKeyHash = PubKeyHash(rewardTxData)
				continue
			case tx.IsFromContract():
				// vinPubKey is the pubKeyHash if it is a sc generated tx
				pubKeyHash = PubKeyHash(vinPubKey)
			default:
				pkh, err := NewUserPubKeyHash(vin.PubKey)
				if err != nil {
					return nil, nil, nil, nil, err
				}
				pubKeyHash = pkh
			}
			usedUTXO := index.FindUTXOByVin([]byte(pubKeyHash), vin.Txid, vin.Vout)
			inputAmount = inputAmount.Add(usedUTXO.Value)
		} else {
			logger.Debug("Transaction: using UTXO from multiple wallets.")
		}
	}
	for _, vout := range tx.Vout {
		if bytes.Compare([]byte(vout.PubKeyHash), vinPubKey) == 0 {
			outputAmount = outputAmount.Add(vout.Value)
		} else {
			receiverAddress = vout.PubKeyHash.GenerateAddress()
			payoutAmount = payoutAmount.Add(vout.Value)
		}
	}
	tip, err := inputAmount.Sub(outputAmount)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	senderAddress := pubKeyHash.GenerateAddress()

	return &senderAddress, &receiverAddress, payoutAmount, tip, nil
}

//GetToHashBytes Get bytes for hash
func (tx *Transaction) GetToHashBytes() []byte {
	var bytes []byte

	for _, vin := range tx.Vin {
		bytes = append(bytes, vin.Txid...)
		// int size may differ from differnt platform
		bytes = append(bytes, byteutils.FromInt32(int32(vin.Vout))...)
		bytes = append(bytes, vin.PubKey...)
		bytes = append(bytes, vin.Signature...)
	}

	for _, vout := range tx.Vout {
		bytes = append(bytes, vout.Value.Bytes()...)
		bytes = append(bytes, []byte(vout.PubKeyHash)...)
		bytes = append(bytes, []byte(vout.Contract)...)
	}
	if tx.Tip != nil {
		bytes = append(bytes, tx.Tip.Bytes()...)
	}
	return bytes
}

// Hash returns the hash of the Transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	hash = sha256.Sum256(tx.GetToHashBytes())

	return hash[:]
}

// Sign signs each input of a Transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevUtxos []*UTXO) error {
	if tx.IsCoinbase() {
		logger.Warn("Transaction: will not sign a coinbase transaction.")
		return nil
	}

	if tx.IsRewardTx() {
		logger.Warn("Transaction: will not sign a reward transaction.")
		return nil
	}

	txCopy := tx.TrimmedCopy(false)
	privData, err := secp256k1.FromECDSAPrivateKey(&privKey)
	if err != nil {
		logger.WithError(err).Error("Transaction: failed to get private key.")
		return err
	}

	for i, vin := range txCopy.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := vin.PubKey
		txCopy.Vin[i].PubKey = []byte(prevUtxos[i].PubKeyHash)
		txCopy.ID = txCopy.Hash()

		txCopy.Vin[i].PubKey = oldPubKey

		signature, err := secp256k1.Sign(txCopy.ID, privData)
		if err != nil {
			logger.WithError(err).Error("Transaction: failed to create a signature.")
			return err
		}

		tx.Vin[i].Signature = signature
	}
	return nil
}

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing
func (tx *Transaction) TrimmedCopy(withSignature bool) Transaction {
	var inputs []TXInput
	var outputs []TXOutput
	var pubkey []byte

	for _, vin := range tx.Vin {
		if withSignature {
			pubkey = vin.PubKey
		} else {
			pubkey = nil
		}
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, pubkey})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash, vout.Contract})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip}

	return txCopy
}

func (tx *Transaction) DeepCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, vin.Signature, vin.PubKey})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash, vout.Contract})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Tip}

	return txCopy
}

func (tx *Transaction) IsContractDeployed(utxos *UTXOIndex) bool {
	contractAddress := tx.GetContractAddress()
	contractUtxos := utxos.GetContractUtxos()
	for _, utxo := range contractUtxos {
		if utxo.PubKeyHash.GenerateAddress() == contractAddress {
			return true
		}
	}
	return false
}

// Verify ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func (tx *Transaction) Verify(utxoIndex *UTXOIndex, blockHeight uint64) bool {
	if tx.IsCoinbase() {
		//TODO coinbase vout check need add tip
		if tx.Vout[0].Value.Cmp(subsidy) < 0 {
			logger.Warn("Transaction: subsidy check failed.")
			return false
		}
		bh := binary.BigEndian.Uint64(tx.Vin[0].Signature)
		if blockHeight != bh {
			logger.Warnf("Transaction: block height check failed expected=%v actual=%v.", blockHeight, bh)
			return false
		}
		return true
	}

	if tx.IsExecutionContract() && !tx.IsContractDeployed(utxoIndex) {
		logger.Warn("Transaction: contract state check failed.")
		return false
	}

	if tx.IsRewardTx() {
		//TODO: verify reward tx here
		return true
	}

	var prevUtxos []*UTXO
	for _, vin := range tx.Vin {
		pubKeyHash, err := NewUserPubKeyHash(vin.PubKey)
		if err != nil {
			logger.WithFields(logger.Fields{
				"tx_id":          hex.EncodeToString(tx.ID),
				"vin_tx_id":      hex.EncodeToString(vin.Txid),
				"vin_public_key": hex.EncodeToString(vin.PubKey),
			}).Warn("Transaction: failed to get PubKeyHash of vin.")
			return false
		}
		utxo := utxoIndex.FindUTXOByVin([]byte(pubKeyHash), vin.Txid, vin.Vout)
		if utxo == nil {
			logger.WithFields(logger.Fields{
				"tx_id":      hex.EncodeToString(tx.ID),
				"vin_tx_id":  hex.EncodeToString(vin.Txid),
				"vin_index":  vin.Vout,
				"pubKeyHash": hex.EncodeToString(pubKeyHash),
			}).Warn("Transaction: cannot find vin.")
			return false
		}
		prevUtxos = append(prevUtxos, utxo)
	}

	if !tx.verifyID() {
		logger.WithFields(logger.Fields{
			"tx_id": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: ID is invalid.")
		return false
	}

	if !tx.verifyPublicKeyHash(prevUtxos) {
		logger.WithFields(logger.Fields{
			"tx_id": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: pubkey is invalid.")
		return false
	}

	if !tx.verifyAmount(prevUtxos) {
		logger.WithFields(logger.Fields{
			"tx_id": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: amount is invalid.")
		return false
	}

	if !tx.verifyTip(prevUtxos) {
		logger.WithFields(logger.Fields{
			"tx_id": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: tip is invalid.")
		return false
	}

	if !tx.verifySignatures(prevUtxos) {
		logger.WithFields(logger.Fields{
			"tx_id": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: signature is invalid.")
		return false
	}

	return true
}

// verifyID verifies if the transaction ID is the hash of the transaction
func (tx *Transaction) verifyID() bool {
	txCopy := tx.TrimmedCopy(true)
	if bytes.Equal(tx.ID, (&txCopy).Hash()) {
		return true
	} else {
		return false
	}
}

//verifyTip verifies if the transaction has the correct tip
func (tx *Transaction) verifyTip(prevUtxos []*UTXO) bool {
	sum := calculateUtxoSum(prevUtxos)
	var err error
	for _, vout := range tx.Vout {
		sum, err = sum.Sub(vout.Value)
		if err != nil {
			return false
		}
	}
	return tx.Tip.Cmp(sum) == 0
}

//verifyPublicKeyHash verifies if the public key in Vin is the original key for the public
//key hash in utxo
func (tx *Transaction) verifyPublicKeyHash(prevUtxos []*UTXO) bool {

	for i, vin := range tx.Vin {

		isContract, err := prevUtxos[i].PubKeyHash.IsContract()
		if err != nil {
			return false
		}
		//if the utxo belongs to a Contract, the utxo is not verified through
		//public key hash. It will be verified through consensus
		if isContract {
			continue
		}

		pubKeyHash, err := NewUserPubKeyHash(vin.PubKey)
		if err != nil {
			return false
		}
		if !bytes.Equal([]byte(pubKeyHash), []byte(prevUtxos[i].PubKeyHash)) {
			return false
		}
	}
	return true
}

func (tx *Transaction) verifySignatures(prevUtxos []*UTXO) bool {
	for _, utxo := range prevUtxos {
		if utxo.PubKeyHash == nil {
			logger.Error("Transaction: previous transaction is not correct.")
			return false
		}
	}

	txCopy := tx.TrimmedCopy(false)

	for i, vin := range tx.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := txCopy.Vin[i].PubKey
		txCopy.Vin[i].PubKey = []byte(prevUtxos[i].PubKeyHash)
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[i].PubKey = oldPubKey

		originPub := make([]byte, 1+len(vin.PubKey))
		originPub[0] = 4 // uncompressed point
		copy(originPub[1:], vin.PubKey)

		verifyResult, err := secp256k1.Verify(txCopy.ID, vin.Signature, originPub)

		if err != nil || verifyResult == false {
			logger.WithError(err).Error("Transaction: signature cannot be verified.")
			return false
		}
	}

	return true
}

func (tx *Transaction) verifyAmount(prevTXs []*UTXO) bool {
	var totalVin, totalVout common.Amount
	for _, utxo := range prevTXs {
		totalVin = *totalVin.Add(utxo.Value)
	}

	for _, vout := range tx.Vout {
		if vout.Value.Validate() != nil {
			return false
		}
		totalVout = *totalVout.Add(vout.Value)
	}
	//TotalVin amount must equal or greater than total vout
	return totalVin.Cmp(&totalVout) >= 0
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to Address, data string, blockHeight uint64, tip *common.Amount) Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}
	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := TXInput{nil, -1, bh, []byte(data)}
	txout := NewTXOutput(subsidy.Add(tip), to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}, common.NewAmount(0)}
	tx.ID = tx.Hash()

	return tx
}

//NewRewardTx creates a new transaction that gives reward to addresses according to the input rewards
func NewRewardTx(blockHeight uint64, rewards map[string]string) Transaction {

	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := TXInput{nil, -1, bh, rewardTxData}
	txOutputs := []TXOutput{}
	for address, amount := range rewards {
		amt, err := common.NewAmountFromString(amount)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"address": address,
				"amount":  amount,
			}).Warn("Transaction: failed to parse reward amount")
		}
		txOutputs = append(txOutputs, *NewTXOutput(amt, NewAddress(address)))
	}
	tx := Transaction{nil, []TXInput{txin}, txOutputs, common.NewAmount(0)}
	tx.ID = tx.Hash()

	return tx
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(utxos []*UTXO, from, to Address, amount *common.Amount, senderKeyPair *KeyPair,
	tip *common.Amount, contract string) (Transaction, error) {

	sum := calculateUtxoSum(utxos)
	change, err := calculateChange(sum, amount, tip)
	if err != nil {
		return Transaction{}, err
	}

	tx := Transaction{
		nil,
		prepareInputLists(utxos, senderKeyPair.PublicKey, nil),
		prepareOutputLists(from, to, amount, change, contract),
		tip}
	tx.ID = tx.Hash()

	err = tx.Sign(senderKeyPair.PrivateKey, utxos)
	if err != nil {
		return Transaction{}, err
	}

	return tx, nil
}

func NewContractTransferTX(utxos []*UTXO, contractAddr, toAddr Address, amount, tip *common.Amount, sourceTXID []byte) (Transaction, error) {
	contractPubKeyHash, ok := contractAddr.GetPubKeyHash()
	if !ok {
		return Transaction{}, ErrInvalidAddress
	}
	if isContract, err := (PubKeyHash(contractPubKeyHash)).IsContract(); !isContract {
		return Transaction{}, err
	}

	sum := calculateUtxoSum(utxos)
	change, err := calculateChange(sum, amount, tip)
	if err != nil {
		return Transaction{}, err
	}

	// Intentionally set PubKeyHash as PubKey (to recognize it is from contract) and sourceTXID as signature in Vin
	tx := Transaction{
		nil,
		prepareInputLists(utxos, contractPubKeyHash, sourceTXID),
		prepareOutputLists(contractAddr, toAddr, amount, change, ""),
		tip,
	}
	tx.ID = tx.Hash()

	return tx, nil
}

func NewTransactionByVin(vinTxId []byte, vinVout int, vinPubkey []byte, voutValue uint64, voutPubKeyHash PubKeyHash, tip uint64) Transaction {
	tx := Transaction{
		ID: nil,
		Vin: []TXInput{
			{vinTxId, vinVout, nil, vinPubkey},
		},
		Vout: []TXOutput{
			{common.NewAmount(voutValue), voutPubKeyHash, ""},
		},
		Tip: common.NewAmount(tip),
	}
	tx.ID = tx.Hash()
	return tx
}

//GetContractAddress gets the smart contract's address if a transaction deploys a smart contract
func (tx *Transaction) GetContractAddress() Address {
	if len(tx.Vout) == 0 {
		return NewAddress("")
	}

	isContract, err := tx.Vout[ContractTxouputIndex].PubKeyHash.IsContract()
	if err != nil {
		return NewAddress("")
	}

	if !isContract {
		return NewAddress("")
	}

	return tx.Vout[ContractTxouputIndex].PubKeyHash.GenerateAddress()
}

//GetContract returns the smart contract code in a transaction
func (tx *Transaction) GetContract() string {
	isContract, _ := tx.Vout[ContractTxouputIndex].PubKeyHash.IsContract()
	if !isContract {
		return ""
	}
	return tx.Vout[ContractTxouputIndex].Contract
}

//Execute executes the smart contract the transaction points to. it doesnt do anything if is a normal transaction
func (tx *Transaction) Execute(index UTXOIndex,
	scStorage *ScState,
	rewards map[string]string,
	engine ScEngine,
	currblkHeight uint64,
	parentBlk *Block) []*Transaction {

	if tx.IsRewardTx() {
		return nil
	}

	vout := tx.Vout[ContractTxouputIndex]

	if isContract, _ := vout.PubKeyHash.IsContract(); !isContract {
		return nil
	}
	utxos := index.GetAllUTXOsByPubKeyHash([]byte(vout.PubKeyHash))
	//the smart contract utxo is always stored at index 0. If there is no utxos found, that means this transaction
	//is a smart contract deployment transaction, not a smart contract execution transaction.
	if len(utxos) == 0 {
		return nil
	}

	prevUtxos, err := tx.FindAllTxinsInUtxoPool(index)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"txid": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: cannot find vin while executing smart contract")
		return nil
	}

	function, args := util.DecodeScInput(vout.Contract)
	if function == "" {
		return nil
	}

	totalArgs := util.PrepareArgs(args)
	address := utxos[0].PubKeyHash.GenerateAddress()
	logger.WithFields(logger.Fields{
		"contract_address": address.String(),
		"invoked_function": function,
		"arguments":        totalArgs,
	}).Debug("Transaction: is executing the smart contract...")
	engine.ImportSourceCode(utxos[0].Contract)
	engine.ImportLocalStorage(scStorage)
	engine.ImportContractAddr(address)
	engine.ImportUTXOs(utxos[1:])
	engine.ImportSourceTXID(tx.ID)
	engine.ImportRewardStorage(rewards)
	engine.ImportTransaction(tx)
	engine.ImportPrevUtxos(prevUtxos)
	engine.ImportCurrBlockHeight(currblkHeight)
	engine.ImportSeed(parentBlk.GetTimestamp())
	engine.Execute(function, totalArgs)
	return engine.GetGeneratedTXs()
}

//FindAllTxinsInUtxoPool Find the transaction in a utxo pool. Returns true only if all Vins are found in the utxo pool
func (tx *Transaction) FindAllTxinsInUtxoPool(utxoPool UTXOIndex) ([]*UTXO, error) {
	var res []*UTXO
	for _, vin := range tx.Vin {
		pubKeyHash, err := NewUserPubKeyHash(vin.PubKey)
		if err != nil {
			return nil, ErrNewUserPubKeyHash
		}
		utxo := utxoPool.FindUTXOByVin([]byte(pubKeyHash), vin.Txid, vin.Vout)
		if utxo == nil {
			MetricsTxDoubleSpend.Inc(1)
			logger.WithFields(logger.Fields{
				"txid":      hex.EncodeToString(tx.ID),
				"vin_id":    hex.EncodeToString(vin.Txid),
				"vin_index": vin.Vout,
			}).Warn("Transaction: Can not find vin")
			return nil, ErrTXInputNotFound
		}
		res = append(res, utxo)
	}
	return res, nil
}

func (tx *Transaction) MatchRewards(rewardStorage map[string]string) bool {

	if tx == nil {
		logger.Debug("Transaction: does not exist")
		return false
	}

	if !tx.IsRewardTx() {
		logger.Debug("Transaction: is not a reward transaction")
		return false
	}

	for _, vout := range tx.Vout {
		if !vout.IsFoundInRewardStorage(rewardStorage) {
			return false
		}
	}
	return len(rewardStorage) == len(tx.Vout)
}

// String returns a human-readable representation of a transaction
func (tx *Transaction) String() string {
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
		lines = append(lines, fmt.Sprintf("       Script: %x", []byte(output.PubKeyHash)))
		lines = append(lines, fmt.Sprintf("       Contract: %s", output.Contract))
	}
	lines = append(lines, "\n")

	return strings.Join(lines, "\n")
}

//calculateChange calculates the change
func calculateChange(input, amount, tip *common.Amount) (*common.Amount, error) {
	change, err := input.Sub(amount)
	if err != nil {
		return nil, ErrInsufficientFund
	}

	change, err = change.Sub(tip)
	if err != nil {
		return nil, ErrInsufficientFund
	}
	return change, nil
}

//prepareInputLists prepares a list of txinputs for a new transaction
func prepareInputLists(utxos []*UTXO, publicKey []byte, signature []byte) []TXInput {
	var inputs []TXInput

	// Build a list of inputs
	for _, utxo := range utxos {
		input := TXInput{utxo.Txid, utxo.TxIndex, signature, publicKey}
		inputs = append(inputs, input)
	}

	return inputs
}

//calculateUtxoSum calculates the total amount of all input utxos
func calculateUtxoSum(utxos []*UTXO) *common.Amount {
	sum := common.NewAmount(0)
	for _, utxo := range utxos {
		sum = sum.Add(utxo.Value)
	}
	return sum
}

//preapreOutPutLists prepares a list of txoutputs for a new transaction
func prepareOutputLists(from, to Address, amount *common.Amount, change *common.Amount, contract string) []TXOutput {

	var outputs []TXOutput
	toAddr := to

	if toAddr.String() == "" {
		toAddr = NewContractPubKeyHash().GenerateAddress()
	}

	if contract != "" {
		outputs = append(outputs, *NewContractTXOutput(toAddr, contract))
	}

	outputs = append(outputs, *NewTXOutput(amount, toAddr))
	if !change.IsZero() {
		outputs = append(outputs, *NewTXOutput(change, from))
	}
	return outputs
}

func (tx *Transaction) ToProto() proto.Message {

	var vinArray []*corepb.TXInput
	for _, txin := range tx.Vin {
		vinArray = append(vinArray, txin.ToProto().(*corepb.TXInput))
	}

	var voutArray []*corepb.TXOutput
	for _, txout := range tx.Vout {
		voutArray = append(voutArray, txout.ToProto().(*corepb.TXOutput))
	}

	return &corepb.Transaction{
		Id:   tx.ID,
		Vin:  vinArray,
		Vout: voutArray,
		Tip:  tx.Tip.Bytes(),
	}
}

func (tx *Transaction) FromProto(pb proto.Message) {
	tx.ID = pb.(*corepb.Transaction).GetId()
	tx.Tip = common.NewAmountFromBytes(pb.(*corepb.Transaction).GetTip())

	var vinArray []TXInput
	txin := TXInput{}
	for _, txinpb := range pb.(*corepb.Transaction).GetVin() {
		txin.FromProto(txinpb)
		vinArray = append(vinArray, txin)
	}
	tx.Vin = vinArray

	var voutArray []TXOutput
	txout := TXOutput{}
	for _, txoutpb := range pb.(*corepb.Transaction).GetVout() {
		txout.FromProto(txoutpb)
		voutArray = append(voutArray, txout)
	}
	tx.Vout = voutArray
}
