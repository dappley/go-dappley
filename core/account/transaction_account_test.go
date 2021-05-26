package account

import (
	"testing"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/stretchr/testify/assert"
)

func TestTransactionAccount_ToProto(t *testing.T) {
	pubKeyHash := newUserPubKeyHash([]byte("address1000000000000000000000000"))
	address := pubKeyHash.GenerateAddress()
	transactionAccount := &TransactionAccount{pubKeyHash: pubKeyHash, address: address}

	expected := &accountpb.TransactionAccount{
		Address: &accountpb.Address{
			Address: address.address,
		},
		PubKeyHash: pubKeyHash,
	}
	assert.Equal(t, transactionAccount.ToProto(), expected)
}

func TestTransactionAccount_FromProto(t *testing.T) {
	pubKeyHash := newUserPubKeyHash([]byte("address1000000000000000000000000"))
	address := pubKeyHash.GenerateAddress()

	transactionAccount := &TransactionAccount{}
	transactionAccountProto := &accountpb.TransactionAccount{
		Address: &accountpb.Address{
			Address: address.address,
		},
		PubKeyHash: pubKeyHash,
	}
	transactionAccount.FromProto(transactionAccountProto)

	expected := &TransactionAccount{address: address, pubKeyHash: pubKeyHash}
	assert.Equal(t, expected, transactionAccount)
}
