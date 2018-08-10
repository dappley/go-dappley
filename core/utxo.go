package core

import (
	"bytes"
	"encoding/gob"
	"log"
	"github.com/dappley/go-dappley/storage"
	"strings"
	"fmt"
	"errors"
	"github.com/jinzhu/copier"
)

//map of key: wallet address, value: serialized map
type utxoIndex map[string][]UTXOutputStored

type UTXOutputStored struct {
	Value      int
	PubKeyHash  []byte
	Txid      []byte
	TxIndex	  int

}


func DeserializeUTXO(d []byte) *utxoIndex {
	var utxo utxoIndex
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&utxo)
	if err != nil {
		fmt.Printf("%+v\n", err.Error())
	}
	return &utxo
}

func (utxo *utxoIndex) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(utxo)
	if err != nil {
		log.Panic(err)
	}
	return encoded.Bytes()
}


func GetAddressUTXOs (mapkey string, pubkey []byte, db storage.Storage ) []UTXOutputStored {
	umap := GetStoredUtxoMap(db, mapkey)
	return umap[string(pubkey)]
}

func GetStoredUtxoMap(db storage.Storage, mapkey string) utxoIndex {
	res, err := db.Get([]byte(mapkey))

	if err != nil && strings.Contains(err.Error(), "Key is invalid") {
		return utxoIndex{}
	}
	umap := DeserializeUTXO(res)
	return *umap
}

func initIndex() utxoIndex {
	ins := map[string][]UTXOutputStored{}
	return  ins
}

func (blk Block) UpdateUtxoIndexAfterNewBlock(mapkey string, db storage.Storage){
	//remove expended outputs
	blk.ConsumeSpendableOutputsAfterNewBlock(mapkey, db)
	//add new outputs
	blk.AddSpendableOutputsAfterNewBlock(mapkey, db)
}

func (blk Block) AddSpendableOutputsAfterNewBlock (mapkey string, db storage.Storage) {
	utxoIndex := GetStoredUtxoMap(db,mapkey)

	if len(utxoIndex)==0 {
		utxoIndex = initIndex()
	}
	for _, txn := range blk.transactions{
		for index ,vout := range txn.Vout{
			if utxoIndex[string(vout.PubKeyHash)] == nil {
				utxoIndex[string(vout.PubKeyHash)] = []UTXOutputStored{}
			}
			utxoIndex[string(vout.PubKeyHash)] = append(utxoIndex[string(vout.PubKeyHash)], UTXOutputStored{vout.Value, vout.PubKeyHash,txn.ID, index})
		}
	}
	db.Put([]byte(UtxoMapKey), utxoIndex.Serialize())
}


func (blk Block) ConsumeSpendableOutputsAfterNewBlock ( mapkey string,db storage.Storage){
	utxoIndex := GetStoredUtxoMap(db,mapkey)
	for _, txns := range blk.transactions{
		for _,vin := range txns.Vin{
			spentOutputTxnId, txnIndex, pubKey := vin.Txid, vin.Vout, string(vin.PubKey)
			userUtxos := utxoIndex[pubKey]
			if(len(userUtxos)) > 0 {
				for index, userUtxo := range userUtxos{
					if(userUtxo.TxIndex == txnIndex && bytes.Compare(userUtxo.Txid,spentOutputTxnId) ==0){
						userUtxos = append(userUtxos[:index], userUtxos[index+1:]...)
					}
				}
				//write to index
				utxoIndex[pubKey] = userUtxos
			}
		}
	}
	utxoIndex.SetUtxoPoolInDb(db)
}

func (utxo *utxoIndex) FindUtxoByTxinput(txin TXInput) *UTXOutputStored{
	for _,utxoArray := range *utxo {
		for _, u := range utxoArray{
			if bytes.Compare(u.Txid,txin.Txid)==0 && u.TxIndex==txin.Vout{
				return &u
			}
		}
	}
	return nil
}

//doesnt save to db
func (utxo utxoIndex) RevertTxnUtxos(blk Block, bc Blockchain, db storage.Storage){

	for _, txn := range blk.GetTransactions() {
		err1:= utxo.RemoveTxnUtxosFromUtxoPool(*txn, db)
		if err1!=nil {
			log.Panic(err1)
		}

		if txn.IsCoinbase(){
			continue
		}

		err2 := utxo.AddBackTxnOutputToUtxoPool(*txn, db, blk, bc)
		if err2!=nil {
			log.Panic(err2)
		}
	}
}

func (utxo utxoIndex) RemoveTxnUtxosFromUtxoPool(txns Transaction, db storage.Storage) error {

	for _,out := range txns.Vout{
		value, pubKey :=  out.Value, string(out.PubKeyHash)
		userUtxos := utxo[pubKey]

		Stud:
			for index, userUtxo := range userUtxos{
				if userUtxo.Value == value {
					//remove utxo from index
					userUtxos = append(userUtxos[:index], userUtxos[index+1:]...)
					break Stud
				}else{
					log.Panic("Address given has no utxos in index")
				}
			}
		utxo[pubKey] = userUtxos
	}
	return nil
}

func (utxo utxoIndex) AddBackTxnOutputToUtxoPool(txn Transaction, db storage.Storage, blk Block, bc Blockchain) error {
	for _, vin := range txn.Vin {
		vout, voutIndex, err := getTXOFromTxIn(vin, blk.GetHash(), bc)
		if err == nil {
			utxo[string(vout.PubKeyHash)] = append(utxo[string(vout.PubKeyHash)], UTXOutputStored{vout.Value, vin.PubKey,txn.ID, voutIndex})
		} else {
			panic(err)
		}
	}
	return nil
}

//set utxopool
func (utxo utxoIndex) SetUtxoPoolInDb(db storage.Storage){
	db.Put([]byte(UtxoMapKey), utxo.Serialize())
}

//block is passed in because i cant statically call FindTransactionById

func getTXOFromTxIn(in TXInput, blkStartIndex []byte, bc Blockchain) (TXOutput, int, error){
	txn, err := bc.FindTransaction(in.Txid)
	if err != nil {
		return  TXOutput{}, 0, errors.New("txInput refers to nonexisting txn")
	}
	return txn.Vout[in.Vout], in.Vout, nil
}


func (utxo utxoIndex) DeepCopy (db storage.Storage) utxoIndex {
	utxocopy := utxoIndex{}
	copier.Copy(&utxo, &utxocopy)
	if len(utxocopy)==0 {
		utxocopy = initIndex()
	}
	return utxocopy
}

//input db and block hash, output utxoindex state @block hash block
func (bc Blockchain) GetUtxoStateAtBlockHash(db storage.Storage, hash []byte) (utxoIndex, error ){
	index := GetStoredUtxoMap(db, UtxoMapKey)
	deepCopy := index.DeepCopy(db)
	bci := bc.Iterator()

	for {
		block, err := bci.Next()

		if bytes.Compare(block.GetHash(), hash) == 0 {
			break
		}

		if err != nil {
			return utxoIndex{}, err
		}

		if len(block.GetPrevHash()) == 0 {
			return utxoIndex{}, ErrBlockDoesNotExist
		}

		deepCopy.RevertTxnUtxos(*block, bc, db)

	}

	return deepCopy, nil
}



