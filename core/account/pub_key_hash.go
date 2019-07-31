package account

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcutil/base58"
	"github.com/dappley/go-dappley/crypto/hash"
)

const versionUser = byte(0x5A)
const versionContract = byte(0x58)
const addressChecksumLen = 4

type PubKeyHash []byte

var (
	ErrIncorrectPublicKey       = errors.New("public key not correct")
	ErrEmptyPublicKeyHash       = errors.New("empty public key hash")
	ErrInvalidPubKeyHashVersion = errors.New("invalid public key hash version")
)

//NewUserPubKeyHash hashes a public key and returns a user type public key hash
func NewUserPubKeyHash(pubKey []byte) (PubKeyHash, error) {
	if ok, err := IsValidPubKey(pubKey); !ok {
		return nil, err
	}
	pubKeyHash := generatePubKeyHash(pubKey)
	pubKeyHash = append([]byte{versionUser}, pubKeyHash...)
	return PubKeyHash(pubKeyHash), nil
}

//NewContractPubKeyHash generates a smart Contract public key hash
func NewContractPubKeyHash() PubKeyHash {
	pubKeyHash := generatePubKeyHash(NewKeyPair().PublicKey)
	pubKeyHash = append([]byte{versionContract}, pubKeyHash...)
	return PubKeyHash(pubKeyHash)
}

//GetPubKeyHash decodes the address to the original public key hash. If unsuccessful, return false
func GeneratePubKeyHashByAddress(a Address) (PubKeyHash, bool) {
	pubKeyHash := base58.Decode(a.String())

	if len(pubKeyHash) != GetAddressPayloadLength() {
		return nil, false
	}
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	pubKeyHash = pubKeyHash[0 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := Checksum(pubKeyHash)

	if bytes.Compare(actualChecksum, targetChecksum) == 0 {
		return pubKeyHash, true
	}
	return nil, false

}

//GenerateAddress generates an address  from a public key hash
func (pkh PubKeyHash) GenerateAddress() Address {
	checksum := Checksum(pkh)
	fullPayload := append(pkh, checksum...)
	return NewAddress(base58.Encode(fullPayload))
}

//GenerateAddress generates an address  from a public key hash
func (pkh PubKeyHash) IsContract() (bool, error) {

	if len(pkh) == 0 {
		return false, ErrEmptyPublicKeyHash
	}

	if pkh[0] == versionUser {
		return false, nil
	}

	if pkh[0] == versionContract {
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
func generatePubKeyHash(pubKey []byte) []byte {
	sha := hash.Sha3256(pubKey)
	content := hash.Ripemd160(sha)
	return content
}

func IsValidPubKey(pubKey []byte) (bool, error) {
	if pubKey == nil || len(pubKey) < 32 {
		return false, ErrIncorrectPublicKey
	}
	return true, nil
}

// GetAddressPayloadLength get the payload length
func GetAddressPayloadLength() int {
	// 1byte(version byte) + 20byte(public key hash bytes) + addressChecksumLen
	return 21 + addressChecksumLen
}

func GetAddressChecksumLen() int {
	return addressChecksumLen
}

func (pkh PubKeyHash) String() string {
	return hex.EncodeToString(pkh)
}

func (pkh PubKeyHash) Equals(npkh PubKeyHash) bool {
	return bytes.Compare(pkh, npkh) == 0
}
