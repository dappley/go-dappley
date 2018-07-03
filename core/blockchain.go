package core

import (
	"log"
	"encoding/hex"
	"fmt"
	"os"
	"bytes"
	"errors"
	"crypto/ecdsa"

	"github.com/dappworks/go-dappworks/storage"
)

const dbFile = "../bin/blockchain.DB"
var tipKey = []byte("1")

type Blockchain struct {
	currentHash []byte
	DB 			*storage.LevelDB
}


// CreateBlockchain creates a new blockchain DB
func CreateBlockchain(address string) *Blockchain {
	if dbExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte
	genesis := NewGenesisBlock(address)

	db, err := storage.OpenDatabase(dbFile)
	if err != nil {
		log.Panic(err)
	}

	err = updateDbWithNewBlock(db, genesis)
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

func GetBlockchain(address string) *Blockchain {
	if dbExists() == false {
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}

	var tip []byte

	db, err := storage.OpenDatabase(dbFile)
	if err != nil {
		log.Panic(err)
	}

	tip, err = db.Get(tipKey)
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

func (bc *Blockchain) MineBlock(transactions []*Transaction) {
	var lastHash []byte

	for _, tx := range transactions {
		if bc.VerifyTransaction(tx) != true {
			//TODO: invalid transaction should be skipped
			log.Panic("ERROR: Invalid transaction")
		}
	}

	lastHash, err := bc.DB.Get(tipKey)

	if err != nil {
		log.Panic(err)
	}

	block := NewBlock(transactions, lastHash)
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.SetHash(hash[:])
	block.SetNonce(nonce)

	err = updateDbWithNewBlock(bc.DB, block)
	if err != nil {
		log.Panic(err)
	}

	bc.currentHash = block.GetHash()

}

//record the new block in the database
func updateDbWithNewBlock(db *storage.LevelDB, newBlock *Block) error{
	err := db.Put(newBlock.GetHash(), newBlock.Serialize())
	if err != nil {
		return err
	}

	err = db.Put(tipKey, newBlock.GetHash())
	if err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

	Work: //TODO
		for _, tx := range unspentTXs {
			txID := hex.EncodeToString(tx.ID)

			for outIdx, out := range tx.Vout {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

					if accumulated >= amount {
						break Work
					}
				}
			}
		}

	return accumulated, unspentOutputs
}

//TODO: optimize performance
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.GetTransactions() {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found")
}

//TODO: optimize performance
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.GetTransactions() {
			txID := hex.EncodeToString(tx.ID)

		Outputs: //TODO
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}

	return unspentTXs
}

func (bc *Blockchain) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}


func (bc *Blockchain) Iterator() *Blockchain {
	return &Blockchain{bc.currentHash, bc.DB}
}

func (bc *Blockchain) Next() *Block {
	var block *Block

	encodedBlock, err := bc.DB.Get(bc.currentHash)
	if err != nil {
		log.Panic(err)
	}

	block = Deserialize(encodedBlock)

	bc.currentHash = block.GetPrevHash()

	return block
}

func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}