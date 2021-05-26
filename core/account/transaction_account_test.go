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
	assert.Equal(t, expected, transactionAccount.ToProto())
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

func TestTransactionAccount_IsValid(t *testing.T) {
	transactionAccount := NewContractTransactionAccount()
	assert.True(t, transactionAccount.IsValid())
	transactionAccount.address.address = "address000000000000000000000000011"
	assert.False(t, transactionAccount.IsValid())
}

func TestNewTransactionAccountByPubKey(t *testing.T) {
	pubKeyBytes := []byte("address1000000000000000000000000")
	transactionAccount := NewTransactionAccountByPubKey(pubKeyBytes)

	assert.NotNil(t, transactionAccount)
	assert.NotNil(t, transactionAccount.pubKeyHash)
	assert.NotNil(t, transactionAccount.address)
	assert.Equal(t, newUserPubKeyHash(pubKeyBytes), transactionAccount.pubKeyHash)
	assert.Equal(t, transactionAccount.pubKeyHash.GenerateAddress(), transactionAccount.address)
}

func TestNewContractAccountByPubKeyHash(t *testing.T) {
	pubKeyBytes := PubKeyHash([]byte{versionUser, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := NewContractAccountByPubKeyHash(pubKeyBytes)

	assert.NotNil(t, transactionAccount)
	assert.NotNil(t, transactionAccount.pubKeyHash)
	assert.NotNil(t, transactionAccount.address)
	assert.Equal(t, pubKeyBytes, transactionAccount.pubKeyHash)
	assert.Equal(t, transactionAccount.pubKeyHash.GenerateAddress(), transactionAccount.address)
}

func TestGeneratePubKeyHashByAddress(t *testing.T) {
	address1 := NewAddress("dZSj3ehsCXKzbTAxfgZU6hokbNFe7Unsuy")
	address2 := NewAddress("invalid000000000000000000000000000")
	address3 := NewAddress("tooshort")
	hash1, success1 := generatePubKeyHashByAddress(address1)
	hash2, success2 := generatePubKeyHashByAddress(address2)
	hash3, success3 := generatePubKeyHashByAddress(address3)
	hash4, success4 := generatePubKeyHashByAddress(address1)

	assert.True(t, success1)
	assert.NotNil(t, hash1)

	assert.False(t, success2)
	assert.Nil(t, hash2)

	assert.False(t, success3)
	assert.Nil(t, hash3)

	assert.True(t, success4)
	assert.Equal(t, hash1, hash4)
}

func TestNewTransactionAccountByAddress(t *testing.T) {
	address := NewAddress("dZSj3ehsCXKzbTAxfgZU6hokbNFe7Unsuy")
	transactionAccount := NewTransactionAccountByAddress(address)
	expectedHash, _ := generatePubKeyHashByAddress(address)

	assert.NotNil(t, transactionAccount)
	assert.NotNil(t, transactionAccount.pubKeyHash)
	assert.NotNil(t, transactionAccount.address)
	assert.Equal(t, expectedHash, transactionAccount.pubKeyHash)
	assert.Equal(t, address, transactionAccount.address)
}

func TestChecksum(t *testing.T) {
	pubKeyBytes1 := []byte{versionUser, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c}
	pubKeyBytes2 := []byte{versionUser, 0xb0, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c}

	checksum1 := Checksum(pubKeyBytes1)
	checksum2 := Checksum(pubKeyBytes2)
	checksum3 := Checksum(pubKeyBytes1)

	assert.Equal(t, addressChecksumLen, len(checksum1))
	assert.Equal(t, addressChecksumLen, len(checksum2))
	assert.NotEqual(t, checksum1, checksum2)
	assert.Equal(t, checksum1, checksum3)
}

func TestGetAddressPayloadLength(t *testing.T) {
	assert.Equal(t, 25, GetAddressPayloadLength())
}