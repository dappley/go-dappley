package core

import (
	"crypto/sha256"
	"errors"

	"github.com/btcsuite/btcutil/base58"
	"github.com/dappley/go-dappley/crypto/hash"
)

const versionUser = byte(0x5A)
const versionContract = byte(0x58)
const addressChecksumLen = 4

type PubKeyHash struct {
	PubKeyHash []byte
}

var (
	ErrIncorrectPublicKey = errors.New("Public key is not correct")
	ErrEmptyPublicKeyHash = errors.New("Empty public key hash")
)

//NewUserPubKeyHash hashes a public key and returns a user type public key hash
func NewUserPubKeyHash(pubKey []byte) (PubKeyHash, error) {
	pubKeyHash, err := generatePubKeyHash(pubKey)
	if err != nil {
		return PubKeyHash{pubKeyHash}, err
	}
	pubKeyHash = append([]byte{versionUser}, pubKeyHash...)
	return PubKeyHash{pubKeyHash}, nil
}

//NewContractPubKeyHash generates a smart Contract public key hash
func NewContractPubKeyHash() PubKeyHash {
	pubKeyHash, _ := generatePubKeyHash(NewKeyPair().PublicKey)
	pubKeyHash = append([]byte{versionContract}, pubKeyHash...)
	return PubKeyHash{pubKeyHash}
}

//GetPubKeyHash gets the public key hash
func (pkh PubKeyHash) GetPubKeyHash() []byte {
	return pkh.PubKeyHash
}

//GenerateAddress generates an address  from a public key hash
func (pkh PubKeyHash) GenerateAddress() Address {
	checksum := Checksum(pkh.GetPubKeyHash())
	fullPayload := append(pkh.GetPubKeyHash(), checksum...)
	return NewAddress(base58.Encode(fullPayload))
}

//GenerateAddress generates an address  from a public key hash
func (pkh PubKeyHash) IsContract() (bool, error) {

	if len(pkh.PubKeyHash) == 0 {
		return false, ErrEmptyPublicKeyHash
	}

	if pkh.PubKeyHash[0] == versionUser {
		return false, nil
	}

	if pkh.PubKeyHash[0] == versionContract {
		return true, nil
	}

	return false, ErrInvalidPubKeyHashVersion
}

//Checksum finds the checksum of a public key hash
func Checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

//generatePubKeyHash hashes a public key
func generatePubKeyHash(pubKey []byte) ([]byte, error) {
	if pubKey == nil || len(pubKey) < 32 {
		return nil, ErrIncorrectPublicKey
	}
	sha := hash.Sha3256(pubKey)
	content := hash.Ripemd160(sha)
	return content, nil
}

// GetAddressPayloadLength get the payload length
func GetAddressPayloadLength() int {
	// 1byte(version byte) + 20byte(public key hash bytes) + addressChecksumLen
	return 21 + addressChecksumLen
}
