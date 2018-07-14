package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"log"

	"fmt"

	"github.com/dappley/go-dappley/storage"
)

var tipKey = []byte("1")

const BlockPoolMaxSize = 100

type Blockchain struct {
	currentHash []byte
	DB          storage.Storage
	blockPool   *BlockPool
	//txPool      *TransactionPool
}

// CreateBlockchain creates a new blockchain DB
func CreateBlockchain(address Address, db storage.Storage) (*Blockchain, error) {
	genesis := NewGenesisBlock(address.Address)
	updateDbWithNewBlock(db, genesis)
	return GetBlockchain(db)
}

func GetBlockchain(db storage.Storage) (*Blockchain, error) {
	var tip []byte
	tip, err := db.Get(tipKey)
	if err != nil {
		return nil, err
	}
	return initializeBlockChainWithBlockPool(tip, db), nil
}

func (bc *Blockchain) UpdateNewBlock(newBlock *Block) {
	updateDbWithNewBlock(bc.DB, newBlock)
	bc.currentHash = newBlock.GetHash()
}

func (bc *Blockchain) BlockPool() *BlockPool {
	return bc.blockPool
}

//func (bc *Blockchain) TransactionPool() *TransactionPool {
//	return bc.txPool
//}
func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int, error) {
	unspentOutputs := make(map[string][]int)
	unspentTXs, err := bc.FindUnspentTransactions(pubKeyHash)
	if err != nil {
		return 0, nil, err
	}
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

	return accumulated, unspentOutputs, nil
}

//TODO: optimize performance
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block, err := bci.Next()
		if err != nil {
			return Transaction{}, err
		}

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
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) ([]Transaction, error) {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block, err := bci.Next()
		if err != nil {
		}

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

	return unspentTXs, nil
}

func (bc *Blockchain) FindUTXO(pubKeyHash []byte) ([]TXOutput, error) {
	var UTXOs []TXOutput
	unspentTransactions, err := bc.FindUnspentTransactions(pubKeyHash)
	if err != nil {
		return nil, err
	}

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs, nil
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

func (bc *Blockchain) VerifyTransaction(tx Transaction) bool {
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
	return initializeBlockChainWithBlockPool(bc.currentHash, bc.DB)
}

func (bc *Blockchain) Next() (*Block, error) {
	var block *Block

	encodedBlock, err := bc.DB.Get(bc.currentHash)
	if err != nil {
		return nil, err
	}

	block = Deserialize(encodedBlock)

	bc.currentHash = block.GetPrevHash()

	return block, nil
}

func (bc *Blockchain) GetLastHash() ([]byte, error) {
	return bc.DB.Get(tipKey)
}

func (bc *Blockchain) String() string {
	var buffer bytes.Buffer

	bci := bc.Iterator()
	for {
		block, err := bci.Next()
		if err != nil {
			fmt.Println(err)
		}

		buffer.WriteString(fmt.Sprintf("============ Block %x ============\n", block.GetHash()))
		buffer.WriteString(fmt.Sprintf("Prev. block: %x\n", block.GetPrevHash()))
		for _, tx := range block.GetTransactions() {
			fmt.Println(tx)
		}
		buffer.WriteString(fmt.Sprintf("\n\n"))

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}
	return buffer.String()
}

func initializeBlockChainWithBlockPool(current []byte, db storage.Storage) *Blockchain {
	blockPool := NewBlockPool(BlockPoolMaxSize)
	return &Blockchain{current, db, blockPool}
}

//record the new block in the database
func updateDbWithNewBlock(db storage.Storage, newBlock *Block) {
	db.Put(newBlock.GetHash(), newBlock.Serialize())

	db.Put(tipKey, newBlock.GetHash())

}
