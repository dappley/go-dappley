package core

import (
	"bytes"
	"encoding/gob"
	"log"
	"github.com/dappley/go-dappley/storage"
)

//map of key: wallet address, value: serialized UTXO minheap
type spendableOutputs map[string]map[string][]byte

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

func getStoredUtxoMap (db storage.LevelDB) spendableOutputs {
	res, err := db.Get([]byte(UtxoMapIndex))
	if err != nil {
		log.Panic(err)
	}
	ins := spendableOutputs{}
	umap := ins.Deserialize(res)
	return *umap
}

// on new txn, outputs will be created which i will need to add to the spendableOutputs map

func AddSpendableOutputsAfterNewBlock (address string, blk Block, db storage.LevelDB) {
	// to be implemented
	//for _, v := range blk.transactions{
	//	for _,va := range v.Vout{
	//		//
	//	}
	//}
}

func ConsumeSpendableOutputs (address string, blk Block, db storage.LevelDB){
	// to be implemented
	//a := []TXOutput{}
	//for _, v := range blk.transactions{
	//	for _,vin := range v.Vin{
	//		txn, err := Blockchain{}.FindTransaction(vin.Txid)
	//		if err != nil {
	//			log.Panic(err)
	//		}
	//
	//	}
	//}

}