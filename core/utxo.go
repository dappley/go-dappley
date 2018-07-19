package core

import (
	"bytes"
	"encoding/gob"
	"log"
	"github.com/dappley/go-dappley/storage"
	"fmt"
)

//map of key: wallet address, value: serialized UTXO minheap
type spendableOutputs map[string]map[string][]TXOutputStored

func (ucache *spendableOutputs) Deserialize(d []byte) *spendableOutputs {
	var txo spendableOutputs
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&txo)
	if err != nil {
		log.Panic(err)
	}
	return &txo
}

func (ucache *spendableOutputs) Serialize() []byte {
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

func GetAddressUTXOs (address string, db storage.LevelDB) (spendableOutputs, error) {
	aob, err := db.Get( []byte(address) )
	if err != nil {
		return nil, err
	}
	ins := spendableOutputs{}
	return *ins.Deserialize(aob), nil
}

func getStoredUtxoMap (db storage.Storage) spendableOutputs {
	res, err := db.Get([]byte(UtxoMapIndex))
	if err != nil {
		log.Panic(err)
	}
	ins := spendableOutputs{}
	umap := ins.Deserialize(res)
	return *umap
}

// on new txn, unspent outputs will be created which i will need to add to the spendableOutputs map

func AddSpendableOutputsAfterNewBlock (blk Block, db storage.Storage) {
	omap := getStoredUtxoMap(db)
	for _, txn := range blk.transactions{
		for index ,va := range txn.Vout{
			omap["utxo"][string(va.PubKeyHash)] = append(omap["utxo"][string(va.PubKeyHash)], TXOutputStored{va.Value, txn.ID, index})
		}
	}
	db.Put([]byte(UtxoMapIndex), omap.Serialize())
}

func ConsumeSpendableOutputsAfterNewBlock (address string, blk Block, db storage.Storage){
	omap := getStoredUtxoMap(db)
	fmt.Printf("%+v\n", omap)
	userUtxos := omap["utxo"][address]
	for _, v := range blk.transactions{
		for _,vin := range v.Vin{
			spentOutputTxnId, txnIndex := vin.Txid, vin.Vout
			for index, userUtxo := range userUtxos{
				if(userUtxo.TxIndex == txnIndex && bytes.Compare(userUtxo.Txid,spentOutputTxnId) ==0){
					//found source utxo, remove from index
					userUtxos = append(userUtxos[:index], userUtxos[index+1:]...)
				}
			}
		}
	}
}
