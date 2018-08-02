package core

import (
	"bytes"
	"encoding/gob"
	"log"
	"github.com/dappley/go-dappley/storage"
	"strings"
	"fmt"
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


func GetAddressUTXOs (pubkey []byte, db storage.Storage, mapkey string) []UTXOutputStored {
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

func UpdateUtxoIndexAfterNewBlock(blk Block, db storage.Storage,mapkey string){
	//remove expended outputs
	ConsumeSpendableOutputsAfterNewBlock(blk, db, mapkey)
	//add new outputs
	AddSpendableOutputsAfterNewBlock(blk, db, mapkey)
}

func AddSpendableOutputsAfterNewBlock (blk Block, db storage.Storage,mapkey string) {
	utxoIndex := GetStoredUtxoMap(db, mapkey)
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

func ConsumeSpendableOutputsAfterNewBlock (blk Block, db storage.Storage, mapkey string){
	utxoIndex := GetStoredUtxoMap(db, mapkey)
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
	SaveToDb(utxoIndex,UtxoMapKey, db)
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
func RevertTxnUtxos(utxoIndex utxoIndex, txn Transaction, blk Block, db storage.Storage){

	RemoveTxnUtxosFromUtxoPool(utxoIndex, txn, db)
	AddBackTxnOutputToTxnPool(txn, db, blk, utxoIndex)
}

func RemoveTxnUtxosFromUtxoPool(utxoIndex utxoIndex, txns Transaction, db storage.Storage) {
	for _,out := range txns.Vout{
		value, pubKey :=  out.Value, string(out.PubKeyHash)
		userUtxos := utxoIndex[pubKey]
		if(len(userUtxos)) > 0 {
			for index, userUtxo := range userUtxos{
				if userUtxo.Value == value {
					//remove utxo from index
					userUtxos = append(userUtxos[:index], userUtxos[index+1:]...)
				}
			}
			utxoIndex[pubKey] = userUtxos
		} else
		{
			panic("No utxos found to work with when reverting utxos")
		}
	}
}

func AddBackTxnOutputToTxnPool(txn Transaction, db storage.Storage, blk Block, utxoIndex utxoIndex) {
	for _, vin := range txn.Vin {
		vout, voutIndex := getTXOFromTxIn(vin, blk)
		utxoIndex[string(vout.PubKeyHash)] = append(utxoIndex[string(vout.PubKeyHash)], UTXOutputStored{vout.Value, vin.PubKey, txn.ID, voutIndex})
	}
}

func SaveToDb(utxoIndex utxoIndex, mapkey string ,db storage.Storage){
	db.Put([]byte(mapkey), utxoIndex.Serialize())
}

//block is passed in because i cant statically call FindTransactionById

func getTXOFromTxIn(in TXInput, blk Block) (TXOutput, int){
	txn := blk.FindTransactionById(in.Txid)
	if txn == nil {
		panic("txInput refers to nonexisting txn")
	}
	return txn.Vout[in.Vout], in.Vout
}

func CreateForkCopyOfUtxoIndex(db storage.Storage) utxoIndex {
	utxoIndex := GetStoredUtxoMap(db, UtxoMapKey)
	if len(utxoIndex)==0 {
		utxoIndex = initIndex()
	}
	SaveToDb(utxoIndex, UtxoForkMapKey, db)
	return utxoIndex
}
