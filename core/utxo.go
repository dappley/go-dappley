package core

import (
	"bytes"
	"encoding/gob"
	"log"
	"github.com/dappley/go-dappley/storage"
	"fmt"
	"strings"
)

//map of key: wallet address, value: serialized map
type spendableOutputs map[string]txoIndex
type txoIndex map[string][]TXOutputStored


func DeserializeUTXO(d []byte) *txoIndex {
	var txo txoIndex
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&txo)
	if err != nil {
		log.Panic(err)
	}
	return &txo
}
func (ucache *txoIndex) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(ucache)
	if err != nil {
		log.Panic(err)
	}
	return encoded.Bytes()
}

func SaveAddressUTXOs (address string, serializedHeap []byte, db storage.LevelDB){
	db.Put( []byte(address), serializedHeap )
}

func GetAddressUTXOs (address []byte, db storage.Storage) []TXOutputStored {
	umap := getStoredUtxoMap(db)
	return umap[string(address)]
}

func getStoredUtxoMap (db storage.Storage) txoIndex {
	res, err := db.Get([]byte(UtxoMapKey))

	if err != nil && strings.Contains(err.Error(), "Key is invalid") {
		print("Map Key is invalid")
		res1 := txoIndex{}
		return res1
	}
	umap := DeserializeUTXO(res)
	return *umap
}

// on new txn, unspent outputs will be created which i will need to add to the spendableOutputs map
func UpdateUtxoIndexAfterNewBlock(blk Block, db storage.Storage){
	AddSpendableOutputsAfterNewBlock(blk, db)
	ConsumeSpendableOutputsAfterNewBlock(blk, db)

}
func AddSpendableOutputsAfterNewBlock (blk Block, db storage.Storage) {
	txoIndex := getStoredUtxoMap(db)
	for _, txn := range blk.transactions{
		for index ,vout := range txn.Vout{
			txoIndex[string(vout.PubKeyHash)] = append(txoIndex[string(vout.PubKeyHash)], TXOutputStored{vout.Value, txn.ID, index})
		}
	}
	db.Put([]byte(UtxoMapKey), txoIndex.Serialize())
}

func ConsumeSpendableOutputsAfterNewBlock (blk Block, db storage.Storage){
	txoIndex := getStoredUtxoMap(db)
	fmt.Printf("%+v\n", txoIndex)

	for _, txns := range blk.transactions{
		for _,vin := range txns.Vin{
			spentOutputTxnId, txnIndex, pubKey := vin.Txid, vin.Vout, string(vin.PubKey)
			userUtxos := txoIndex[pubKey]
			for index, userUtxo := range userUtxos{
				if(userUtxo.TxIndex == txnIndex && bytes.Compare(userUtxo.Txid,spentOutputTxnId) ==0){
					userUtxos = append(userUtxos[:index], userUtxos[index+1:]...)
				}
			}
		}
	}
}
