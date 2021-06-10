package account

import (
	"testing"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/stretchr/testify/assert"
)

func TestTransactionAccount_ToProto(t *testing.T) {
	transactionAccount := &TransactionAccount{
		Address{ "cd2MRu285Uwiu8ZkDp4jtL8tcZeHMZk8YL"},
		[]byte{88,134, 181, 86, 183, 18, 242, 27, 204, 7, 217, 60, 186, 131, 186, 176, 222, 153, 72, 62, 0},
	}

	expected := &accountpb.TransactionAccount{
		Address: &accountpb.Address{
			Address: "cd2MRu285Uwiu8ZkDp4jtL8tcZeHMZk8YL",
		},
		PubKeyHash: []byte{88,134, 181, 86, 183, 18, 242, 27, 204, 7, 217, 60, 186, 131, 186, 176, 222, 153, 72, 62, 0},
	}

	assert.Equal(t, expected, transactionAccount.ToProto())
}

func TestTransactionAccount_FromProto(t *testing.T) {

	transactionAccount := &TransactionAccount{}
	transactionAccountProto := &accountpb.TransactionAccount{
		Address: &accountpb.Address{
			Address: 		"cd2MRu285Uwiu8ZkDp4jtL8tcZeHMZk8YL",
		},
		PubKeyHash: []byte{88,134, 181, 86, 183, 18, 242, 27, 204, 7, 217, 60, 186, 131, 186, 176, 222, 153, 72, 62, 0},
	}
	transactionAccount.FromProto(transactionAccountProto)

	expected  := &TransactionAccount{
		Address{ "cd2MRu285Uwiu8ZkDp4jtL8tcZeHMZk8YL"},
		[]byte{88,134, 181, 86, 183, 18, 242, 27, 204, 7, 217, 60, 186, 131, 186, 176, 222, 153, 72, 62, 0},
	}
	assert.Equal(t, expected, transactionAccount)
}

func TestTransactionAccount_IsValid(t *testing.T) {
	transactionAccount := &TransactionAccount{
		Address{ "cd2MRu285Uwiu8ZkDp4jtL8tcZeHMZk8YL"},
		[]byte{88,134, 181, 86, 183, 18, 242, 27, 204, 7, 217, 60, 186, 131, 186, 176, 222, 153, 72, 62, 0},
	}
	assert.True(t, transactionAccount.IsValid())

	transactionAccount.pubKeyHash=[]byte{}
	assert.False(t, transactionAccount.IsValid())

	transactionAccount.pubKeyHash = []byte{88,134, 181, 86, 183, 18, 242, 27, 204, 7, 217, 60, 186, 131, 186, 176, 222, 153, 72, 62, 0}
	transactionAccount.address.address = "address000000000000000000000000011"
	assert.False(t, transactionAccount.IsValid())
}

func TestNewTransactionAccountByPubKey(t *testing.T) {
	pubKeyBytes := []byte("address1000000000000000000000000")
	transactionAccount := NewTransactionAccountByPubKey(pubKeyBytes)

	assert.Equal(t, PubKeyHash([]byte{0x5a, 0xad, 0xec, 0x2c, 0x21, 0x3b, 0x67, 0xfa, 0x96, 0xe5, 0xa8, 0xb9, 0xb4, 0x99, 0xf3, 0x26, 0x41, 0xf7, 0xff, 0x36, 0x8a}), transactionAccount.pubKeyHash)
	assert.Equal(t, Address{address: "dVGuSWFXE91Ay36n9HnCzpu8AfckEgvnnR"}, transactionAccount.address)
}

func TestNewContractAccountByPubKeyHash(t *testing.T) {
	pubKeyBytes := PubKeyHash([]byte{versionUser, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := NewContractAccountByPubKeyHash(pubKeyBytes)

	assert.Equal(t, pubKeyBytes, transactionAccount.pubKeyHash)
	assert.Equal(t, Address{address: "dVaFsQL9He4Xn4CEUh1TCNtfEhHNHKX3hs"}, transactionAccount.address)
}

func TestGeneratePubKeyHashByAddress(t *testing.T) {
	address := Address{address: "dZSj3ehsCXKzbTAxfgZU6hokbNFe7Unsuy"}
	invalidAddress := Address{address: "invalid000000000000000000000000000"}
	tooshortAddress := Address{address: "tooshort"}
	hash1, result1 := generatePubKeyHashByAddress(address)
	hash2, result2 := generatePubKeyHashByAddress(invalidAddress)
	hash3, result3 := generatePubKeyHashByAddress(tooshortAddress)

	expectedHash1 := PubKeyHash([]byte{0x5a, 0xdb, 0xa8, 0x28, 0x9b, 0xe2, 0xa9, 0xf, 0x21, 0x1f, 0xf5, 0x0, 0x5f, 0x2a, 0x8e, 0x1e, 0xe8, 0x90, 0x62, 0x5c, 0x2})

	assert.True(t, result1)
	assert.Equal(t, expectedHash1, hash1)

	assert.False(t, result2)
	assert.Nil(t, hash2)

	assert.False(t, result3)
	assert.Nil(t, hash3)
}

func TestNewTransactionAccountByAddress(t *testing.T) {
	address := Address{address: "dZSj3ehsCXKzbTAxfgZU6hokbNFe7Unsuy"}
	transactionAccount := NewTransactionAccountByAddress(address)
	expectedHash := PubKeyHash([]byte{0x5a, 0xdb, 0xa8, 0x28, 0x9b, 0xe2, 0xa9, 0xf, 0x21, 0x1f, 0xf5, 0x0, 0x5f, 0x2a, 0x8e, 0x1e, 0xe8, 0x90, 0x62, 0x5c, 0x2})

	assert.Equal(t, expectedHash, transactionAccount.pubKeyHash)
	assert.Equal(t, address, transactionAccount.address)
}

func TestChecksum(t *testing.T) {
	pubKeyBytes1 := []byte{versionUser, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c}
	pubKeyBytes2 := []byte{versionContract, 0xb0, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c}

	assert.Equal(t, []byte{0x8d, 0xc6, 0x1e, 0x9a}, Checksum(pubKeyBytes1))
	assert.Equal(t, []byte{0xf9, 0x2d, 0xf1, 0x40}, Checksum(pubKeyBytes2))
}

func TestGetAddressPayloadLength(t *testing.T) {
	assert.Equal(t, 25, GetAddressPayloadLength())
}