package core


type UTXOutputStored struct {
	Value      	int
	PubKeyHash 	[]byte
	Txid      	[]byte
	TxIndex	  	int
}

