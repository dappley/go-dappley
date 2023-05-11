package transaction

import (
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type Transactions struct {
	transactions []Transaction
}

type NonceTransactions struct {
	transactions []NonceTransaction
}

type NonceTransaction struct {
	transaction *Transaction
	nonce       uint64
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

func NewNonceTransaction(tx *Transaction, nonce uint64) NonceTransaction {
	return NonceTransaction{tx, nonce}
}

func (tx *NonceTransaction) ToProto() proto.Message {
	return &transactionpb.NonceTransaction{
		Transaction: tx.ToProto().(*transactionpb.Transaction),
		Nonce:       tx.nonce,
	}
}

func (tx *NonceTransaction) FromProto(pb proto.Message) {
	txpb := pb.(*transactionpb.NonceTransaction)
	baseTx := &Transaction{}
	baseTx.FromProto(txpb.GetTransaction())
	tx.transaction = baseTx
	tx.nonce = txpb.GetNonce()
}

func (tx *NonceTransaction) GetTransaction() *Transaction {
	return tx.transaction
}

func (tx *NonceTransaction) GetNonce() uint64 {
	return tx.nonce
}

func NewNonceTransactions(txs []Transaction, nonces []uint64) *NonceTransactions {
	if len(txs) != len(nonces) {
		logger.Warn("each tx should have an associated nonce")
		return &NonceTransactions{}
	}
	nonceTxs := []NonceTransaction{}
	for i, tx := range txs {
		nonceTxs = append(nonceTxs, NonceTransaction{&tx, nonces[i]})
	}
	return &NonceTransactions{nonceTxs}
}

func (txs *NonceTransactions) GetTransactions() []NonceTransaction {
	return txs.transactions
}

func (txs *NonceTransactions) ToProto() proto.Message {

	var txspb []*transactionpb.NonceTransaction
	for _, tx := range txs.transactions {
		txspb = append(txspb, tx.ToProto().(*transactionpb.NonceTransaction))
	}

	return &transactionpb.NonceTransactions{
		Transactions: txspb,
	}
}

func (txs *NonceTransactions) FromProto(pb proto.Message) {
	var transactions []NonceTransaction

	for _, txpb := range pb.(*transactionpb.NonceTransactions).GetTransactions() {
		tx := NonceTransaction{}
		tx.FromProto(txpb)
		transactions = append(transactions, tx)
	}
	txs.transactions = transactions
}
