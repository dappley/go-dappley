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


func GetAddressUTXOs (pubkey []byte, db storage.Storage) []UTXOutputStored {
	umap := getStoredUtxoMap(db)
	return umap[string(pubkey)]
}

func getStoredUtxoMap (db storage.Storage) utxoIndex {
	res, err := db.Get([]byte(UtxoMapKey))

	if err != nil && strings.Contains(err.Error(), "Key is invalid") {
		res1 := utxoIndex{}
		return res1
	}
	umap := DeserializeUTXO(res)
	return *umap
}

func initIndex() utxoIndex {
	ins := map[string][]UTXOutputStored{}
	return  ins
}

func UpdateUtxoIndexAfterNewBlock(blk Block, db storage.Storage){
	//remove expended outputs
	ConsumeSpendableOutputsAfterNewBlock(blk, db)
	//add new outputs
	AddSpendableOutputsAfterNewBlock(blk, db)


}
func AddSpendableOutputsAfterNewBlock (blk Block, db storage.Storage) {
	utxoIndex := getStoredUtxoMap(db)
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

func ConsumeSpendableOutputsAfterNewBlock (blk Block, db storage.Storage){
	utxoIndex := getStoredUtxoMap(db)
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
				utxoIndex[pubKey] = userUtxos
			}
		}
	}
	db.Put([]byte(UtxoMapKey), utxoIndex.Serialize())
}

func (utxo *utxoIndex) VerifyTransactionInput(txin TXInput) bool{
	for _,utxoArray := range *utxo {
		for _, u := range utxoArray{
			if bytes.Compare(u.Txid,txin.Txid)==0 && u.TxIndex==txin.Vout{
				return true
			}
		}
	}
	return false
}