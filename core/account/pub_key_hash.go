package account

import (
	"encoding/hex"

	"github.com/btcsuite/btcutil/base58"
	"github.com/dappley/go-dappley/crypto/hash"
	errorValues "github.com/dappley/go-dappley/errors"
)

const versionUser = byte(0x5A)
const versionContract = byte(0x58)
const addressChecksumLen = 4

type PubKeyHash []byte

//NewUserPubKeyHash hashes a public key and returns a user type public key hash
func newUserPubKeyHash(pubKey []byte) PubKeyHash {
	pubKeyHash := generatePubKeyHash(pubKey)
	pubKeyHash = append([]byte{versionUser}, pubKeyHash...)
	return PubKeyHash(pubKeyHash)
}

//NewContractPubKeyHash generates a smart Contract public key hash
func newContractPubKeyHash() PubKeyHash {
	pubKeyHash := generatePubKeyHash(NewKeyPair().GetPublicKey())
	pubKeyHash = append([]byte{versionContract}, pubKeyHash...)
	return PubKeyHash(pubKeyHash)
}
func (pkh PubKeyHash) IsValid() bool {
	if len(pkh) != 21 {
		return false
	}
	return true
}

//GenerateAddress generates an address  from a public key hash
func (pkh PubKeyHash) GenerateAddress() Address {
	checksum := Checksum(pkh)
	fullPayload := append(pkh, checksum...)
	return NewAddress(base58.Encode(fullPayload))
}

//IsContract return true if it is a contract address
func (pkh PubKeyHash) IsContract() (bool, error) {

	if len(pkh) == 0 {
		return false, errorValues.ErrEmptyPublicKeyHash
	}

	if pkh[0] == versionUser {
		return false, nil
	}

	if pkh[0] == versionContract {
		return true, nil
	}

	return false, errorValues.ErrInvalidPubKeyHashVersion
}

//generatePubKeyHash hashes a public key
func generatePubKeyHash(pubKey []byte) []byte {
	sha := hash.Sha3256(pubKey)
	content := hash.Ripemd160(sha)
	return content
}

//IsValidPubKey return true if pubkey is valid
func IsValidPubKey(pubKey []byte) (bool, error) {
	if pubKey == nil || len(pubKey) < 32 {
		return false, errorValues.ErrIncorrectPublicKey
	}
	return true, nil
}

func (pkh PubKeyHash) String() string {
	return hex.EncodeToString(pkh)
}
