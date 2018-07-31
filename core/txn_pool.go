package core

import (
	"sync"
	"github.com/dappley/go-dappley/common/sorted"
	"bytes"
	"fmt"
)

type TransactionPoool struct {
	messageCh    chan string
	exitCh       chan bool
	size         int
	transactions sorted.Slice
}

var instance23812531 *TransactionPoool
var once23812531 sync.Once


func CompareTransactionTips(a interface{}, b interface{}) int {
	ai := a.(Transaction)
	bi := b.(Transaction)
	if ai.Tip < bi.Tip {
		return -1
	} else if ai.Tip > bi.Tip {
		return 1
	} else {
		return 0
	}
}

func (s *TransactionPoool) StructDelete(txn interface{}) {
	for k, v := range s.transactions.GetSliceContent() {
		if bytes.Compare(v.(Transaction).ID, txn.(Transaction).ID) == 0  {

			var content []interface{}
			content = append(content, s.transactions.GetSliceContent()[k+1:]...)
			content = append(s.transactions.GetSliceContent()[0:k], content...)
			s.transactions.SetSliceContent(content)
			return
		}
	}
}
// Push a new value into slice
func (s *TransactionPoool) StructPush(val interface{}) {
	if s.transactions.Len() == 0 {
		 s.transactions.AddSliceItem(val)
		return
	}

	start, end := 0, s.transactions.Len()-1
	result, mid := 0, 0
	for start <= end {
		mid = (start + end) / 2
		cmp:= s.transactions.GetSliceCmp()
		result = cmp(s.transactions.Index(mid), val)
		if result > 0 {
			end = mid - 1
		} else if result < 0 {
			start = mid + 1
		} else {
			break
		}
	}
	content := []interface{}{val}
	if result > 0 {
		content = append(content, s.transactions.GetSliceContent()[mid:]...)
		content = append(s.transactions.GetSliceContent()[0:mid], content...)
	} else {
		content = append(content, s.transactions.GetSliceContent()[mid+1:]...)
		content = append(s.transactions.GetSliceContent()[0:mid+1], content...)

	}
	s.transactions.SetSliceContent(content)
}

func GetTxnPoolInstance23812531() *TransactionPoool {
	once23812531.Do(func() {
		//instance = &TransactionPool{}
		instance23812531 = &TransactionPoool{
			messageCh: make(chan string, 128),
			size:      128,
		}
	})

	instance23812531.transactions = *sorted.NewSlice(CompareTransactionTips,instance23812531.StructDelete, instance23812531.StructPush)

	return instance23812531
}