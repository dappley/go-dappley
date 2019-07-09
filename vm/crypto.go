package vm

import "C"
import (
	"crypto/sha256"
	"encoding/hex"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
)

//export VerifySignatureFunc
func VerifySignatureFunc(msg, pubkey, sig *C.char) bool {
	goMsg := C.GoString(msg)
	goPubkey := C.GoString(pubkey)
	goSig := C.GoString(sig)

	data := sha256.Sum256([]byte(goMsg))

	sigBytes, err := hex.DecodeString(goSig)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"content":   goMsg,
			"signature": sigBytes,
		}).Debug("SmartContract: failed to decode signature.")
		return false
	}

	pubKeyBytes, err := hex.DecodeString(goPubkey)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"content":    goMsg,
			"public_key": pubKeyBytes,
		}).Debug("SmartContract: failed to decode public key.")
		return false
	}

	originPub := make([]byte, 1+len(pubKeyBytes))
	originPub[0] = 4 // uncompressed point
	copy(originPub[1:], pubKeyBytes)

	res, err := secp256k1.Verify(data[:], sigBytes, originPub)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"content":    goMsg,
			"public_key": pubKeyBytes,
			"signature":  sigBytes,
		}).Debug("SmartContract: failed to verify signature.")
		return false
	}

	return res
}

//export VerifyPublicKeyFunc
func VerifyPublicKeyFunc(addr, pubkey *C.char) bool {
	goAddr := C.GoString(addr)
	goPubkey := C.GoString(pubkey)

	pubKeyBytes, err := hex.DecodeString(goPubkey)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"address":    goAddr,
			"public_key": pubKeyBytes,
		}).Debug("SmartContract: failed to decode public key.")
		return false
	}

	pubKeyHash, err := core.NewUserPubKeyHash(pubKeyBytes)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"content":    goAddr,
			"public_key": pubKeyBytes,
		}).Debug("SmartContract: failed to hash public key.")
		return false
	}

	return pubKeyHash.GenerateAddress().String() == goAddr
}
