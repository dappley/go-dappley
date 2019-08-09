package block_logic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"

	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/transaction_logic"
	"github.com/dappley/go-dappley/logic/utxo_logic"

	"github.com/dappley/go-dappley/vm"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/crypto/sha3"
	"github.com/dappley/go-dappley/util"
	logger "github.com/sirupsen/logrus"
)

func HashTransactions(b *block.Block) []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.GetTransactions() {
		txHashes = append(txHashes, tx.Hash())
	}
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}

func CalculateHash(b *block.Block) hash.Hash {
	return CalculateHashWithNonce(b)
}

func CalculateHashWithoutNonce(b *block.Block) hash.Hash {
	data := bytes.Join(
		[][]byte{
			b.GetPrevHash(),
			HashTransactions(b),
			util.IntToHex(b.GetTimestamp()),
			[]byte(b.GetProducer()),
		},
		[]byte{},
	)

	hasher := sha3.New256()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func CalculateHashWithNonce(b *block.Block) hash.Hash {
	data := bytes.Join(
		[][]byte{
			b.GetPrevHash(),
			HashTransactions(b),
			util.IntToHex(b.GetTimestamp()),
			//util.IntToHex(targetBits),
			util.IntToHex(b.GetNonce()),
			[]byte(b.GetProducer()),
		},
		[]byte{},
	)
	h := sha256.Sum256(data)
	return h[:]
}

func SignBlock(b *block.Block, key string) bool {
	if len(key) <= 0 {
		logger.Warn("Block: the key is too short for signature!")
		return false
	}

	signature, err := generateSignature(key, b.GetHash())
	if err != nil {
		return false
	}
	b.SetSignature(signature)
	return true
}

func generateSignature(key string, data hash.Hash) (hash.Hash, error) {
	privData, err := hex.DecodeString(key)

	if err != nil {
		logger.Warn("Block: cannot decode private key for signature!")
		return []byte{}, err
	}
	signature, err := secp256k1.Sign(data, privData)
	if err != nil {
		logger.WithError(err).Warn("Block: failed to calculate signature!")
		return []byte{}, err
	}

	return signature, nil
}

func VerifyHash(b *block.Block) bool {
	return bytes.Compare(b.GetHash(), CalculateHash(b)) == 0
}

func VerifyTransactions(b *block.Block, utxoIndex *utxo_logic.UTXOIndex, scState *scState.ScState, parentBlk *block.Block) bool {
	if len(b.GetTransactions()) == 0 {
		logger.WithFields(logger.Fields{
			"hash": b.GetHash(),
		}).Debug("Block: there is no transaction to verify in this block.")
		return true
	}

	var rewardTX *transaction.Transaction
	var contractGeneratedTXs []*transaction.Transaction
	rewards := make(map[string]string)
	var allContractGeneratedTXs []*transaction.Transaction

	scEngine := vm.NewV8Engine()
	defer scEngine.DestroyEngine()

L:
	for _, tx := range b.GetTransactions() {
		// Collect the contract-incurred transactions in this block
		if tx.IsRewardTx() {
			if rewardTX != nil {
				logger.Warn("Block: contains more than 1 reward transaction.")
				return false
			}
			rewardTX = tx
			utxoIndex.UpdateUtxo(tx)
			continue L
		}
		if transaction_logic.IsFromContract(utxoIndex, tx) {
			contractGeneratedTXs = append(contractGeneratedTXs, tx)
			continue L
		}

		ctx := tx.ToContractTx()
		if ctx != nil {
			// Run the contract and collect generated transactions
			if scEngine == nil {
				logger.Warn("Block: smart contract cannot be verified.")
				logger.Debug("Block: is missing SCEngineManager when verifying transactions.")
				return false
			}

			prevUtxos, err := utxo_logic.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
			if err != nil {
				logger.WithError(err).WithFields(logger.Fields{
					"txid": hex.EncodeToString(ctx.ID),
				}).Warn("Transaction: cannot find vin while executing smart contract")
				return false
			}

			isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0
			// TODO GAS LIMIT
			if err := scEngine.SetExecutionLimits(1000, 0); err != nil {
				return false
			}
			transaction_logic.Execute(ctx, prevUtxos, isSCUTXO, *utxoIndex, scState, rewards, scEngine, b.GetHeight(), parentBlk)
			allContractGeneratedTXs = append(allContractGeneratedTXs, scEngine.GetGeneratedTXs()...)
		} else {
			// tx is a normal transactions
			if result, err := transaction_logic.VerifyTransaction(utxoIndex, tx, b.GetHeight()); !result {
				logger.Warn(err.Error())
				return false
			}
			utxoIndex.UpdateUtxo(tx)
		}
	}
	// Assert that any contract-incurred transactions matches the ones generated from contract execution
	if rewardTX != nil && !rewardTX.MatchRewards(rewards) {
		logger.Warn("Block: reward tx cannot be verified.")
		return false
	}
	if len(contractGeneratedTXs) > 0 && !verifyGeneratedTXs(utxoIndex, contractGeneratedTXs, allContractGeneratedTXs) {
		logger.Warn("Block: generated tx cannot be verified.")
		return false
	}
	utxoIndex.UpdateUtxoState(allContractGeneratedTXs)
	return true
}

// verifyGeneratedTXs verify that all transactions in candidates can be found in generatedTXs
func verifyGeneratedTXs(utxoIndex *utxo_logic.UTXOIndex, candidates []*transaction.Transaction, generatedTXs []*transaction.Transaction) bool {
	// genTXBuckets stores description of txs grouped by concatenation of sender's and recipient's public key hashes
	genTXBuckets := make(map[string][][]*common.Amount)
	for _, genTX := range generatedTXs {
		sender, recipient, amount, tip, err := transaction_logic.DescribeTransaction(utxoIndex, genTX)
		if err != nil {
			continue
		}
		hashKey := sender.String() + recipient.String()
		genTXBuckets[hashKey] = append(genTXBuckets[hashKey], []*common.Amount{amount, tip})
	}
L:
	for _, tx := range candidates {
		sender, recipient, amount, tip, err := transaction_logic.DescribeTransaction(utxoIndex, tx)
		if err != nil {
			return false
		}
		hashKey := sender.String() + recipient.String()
		if genTXBuckets[hashKey] == nil {
			return false
		}
		for i, t := range genTXBuckets[hashKey] {
			// tx is verified if amount and tip matches
			if amount.Cmp(t[0]) == 0 && tip.Cmp(t[1]) == 0 {
				genTXBuckets[hashKey] = append(genTXBuckets[hashKey][:i], genTXBuckets[hashKey][i+1:]...)
				continue L
			}
		}
		return false
	}
	return true
}

func IsHashEqual(h1 hash.Hash, h2 hash.Hash) bool {

	return reflect.DeepEqual(h1, h2)
}
