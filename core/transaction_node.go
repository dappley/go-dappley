package core

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/golang/protobuf/proto"
)

type TransactionNode struct {
	Children map[string]*transaction.Transaction
	Value    *transaction.Transaction
	Size     int
}

func NewTransactionNode(tx *transaction.Transaction) *TransactionNode {
	txNode := &TransactionNode{Children: make(map[string]*transaction.Transaction)}

	if tx == nil {
		return txNode
	}

	size := tx.GetSize()
	if size == 0 {
		return txNode
	}

	txNode.Value = tx
	txNode.Size = size

	return txNode
}

func (txNode *TransactionNode) GetTipsPerByte() *common.Amount {
	return txNode.Value.Tip.Times(uint64(100000)).Div(uint64(txNode.Size))
}

func (txNode *TransactionNode) ToProto() proto.Message {
	childrenProto := make(map[string]*transactionpb.Transaction)
	for key, val := range txNode.Children {
		childrenProto[key] = val.ToProto().(*transactionpb.Transaction)
	}
	return &corepb.TransactionNode{
		Children: childrenProto,
		Value:    txNode.Value.ToProto().(*transactionpb.Transaction),
		Size:     int64(txNode.Size),
	}
}

func (txNode *TransactionNode) FromProto(pb proto.Message) {
	for key, val := range pb.(*corepb.TransactionNode).Children {
		tx := &transaction.Transaction{}
		tx.FromProto(val)
		txNode.Children[key] = tx
	}
	tx := &transaction.Transaction{}
	tx.FromProto(pb.(*corepb.TransactionNode).Value)
	txNode.Value = tx
	txNode.Size = int(pb.(*corepb.TransactionNode).Size)
}
