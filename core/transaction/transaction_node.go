package transaction

import (
	"github.com/dappley/go-dappley/common"
)

type TransactionNode struct {
	Children map[string]*Transaction
	Value    *Transaction
	Size     int
	Nonce    uint64
}

func NewTransactionNode(tx *Transaction, nonce uint64) *TransactionNode {
	txNode := &TransactionNode{Children: make(map[string]*Transaction)}

	if tx == nil {
		return txNode
	}

	size := tx.GetSize()
	if size == 0 {
		return txNode
	}

	txNode.Value = tx
	txNode.Size = size
	txNode.Nonce = nonce

	return txNode
}

func (txNode *TransactionNode) GetTipsPerByte() *common.Amount {
	return txNode.Value.Tip.Times(uint64(100000)).Div(uint64(txNode.Size))
}
