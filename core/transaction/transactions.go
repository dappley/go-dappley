package transaction

import (
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/golang/protobuf/proto"
)

type Transactions struct {
	transactions []Transaction
}

func NewTransactions(txs []Transaction) *Transactions {
	return &Transactions{txs}
}

func (txs *Transactions) GetTransactions() []Transaction {
	return txs.transactions
}

func (txs *Transactions) ToProto() proto.Message {

	var txspb []*transactionpb.Transaction
	for _, tx := range txs.transactions {
		txspb = append(txspb, tx.ToProto().(*transactionpb.Transaction))
	}

	return &transactionpb.Transactions{
		Transactions: txspb,
	}
}

func (txs *Transactions) FromProto(pb proto.Message) {
	var transactions []Transaction

	for _, txpb := range pb.(*transactionpb.Transactions).GetTransactions() {
		tx := Transaction{}
		tx.FromProto(txpb)
		transactions = append(transactions, tx)
	}
	txs.transactions = transactions
}
