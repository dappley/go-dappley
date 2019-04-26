package core

import (
	"github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"
)

type Transactions struct {
	transactions []*Transaction
}

func NewTransactions(txs []*Transaction) *Transactions {
	return &Transactions{txs}
}

func (txs *Transactions) ToProto() proto.Message {

	var txspb []*corepb.Transaction
	for _, tx := range txs.transactions {
		txspb = append(txspb, tx.ToProto().(*corepb.Transaction))
	}

	return &corepb.Transactions{
		Transactions: txspb,
	}
}

func (txs *Transactions) FromProto(pb proto.Message) {
	var transactions []*Transaction

	for _, txpb := range pb.(*corepb.Transactions).GetTransactions() {
		tx := &Transaction{}
		tx.FromProto(txpb)
		transactions = append(transactions, tx)
	}
	txs.transactions = transactions
}
