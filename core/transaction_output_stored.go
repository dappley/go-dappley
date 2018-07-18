package core


type TXOutputStored struct {
	Value      int
	PubKeyHash []byte
	Txid      []byte
	TxIndex	  int
}

