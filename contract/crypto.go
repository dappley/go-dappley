package vm
import "C"
import (
	"encoding/hex"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	logger "github.com/sirupsen/logrus"
)

//export VerifySignatureFunc
func VerifySignatureFunc(msg, pubkey, sig *C.char) bool{
	goMsg := C.GoString(msg)
	goPubkey := C.GoString(pubkey)
	goSig := C.GoString(sig)

	msgBytes, err := hex.DecodeString(goMsg)
	if err!= nil{
		logger.WithFields(logger.Fields{
			"content"	: goMsg,
			"error"		: err,
		}).Debug("Smart Contract: VerifySignature failed. Unable to decode message")
		return false
	}

	sigBytes, err := hex.DecodeString(goSig)
	if err!= nil{
		logger.WithFields(logger.Fields{
			"content"	: goMsg,
			"Signature" : sigBytes,
			"error" 	: err,
		}).Debug("Smart Contract: VerifySignature failed. Unable to decode signature")
		return false
	}

	pubKeyBytes, err :=	hex.DecodeString(goPubkey)
	if err!= nil{
		logger.WithFields(logger.Fields{
			"content"	: goMsg,
			"pubKey" 	: pubKeyBytes,
			"error" 	: err,
		}).Debug("Smart Contract: VerifySignature failed. Unable to decode public key")
		return false
	}

	originPub := make([]byte, 1+len(pubKeyBytes))
	originPub[0] = 4 // uncompressed point
	copy(originPub[1:], pubKeyBytes)

	res, err := secp256k1.Verify(msgBytes, sigBytes, originPub)
	if err!= nil{
		logger.WithFields(logger.Fields{
			"content"	: goMsg,
			"pubKey" 	: pubKeyBytes,
			"Signature" : sigBytes,
			"error" 	: err,
		}).Debug("Smart Contract: VerifySignature failed.")
		return false
	}

	return res
}

//export VerifyPublicKeyFunc
func VerifyPublicKeyFunc(addr, pubkey *C.char) bool{
	goAddr := C.GoString(addr)
	goPubkey := C.GoString(pubkey)

	pubKeyBytes, err :=	hex.DecodeString(goPubkey)
	if err!= nil{
		logger.WithFields(logger.Fields{
			"address"	: goAddr,
			"pubKey" 	: pubKeyBytes,
			"error" 	: err,
		}).Debug("Smart Contract: VerifyPublicKey failed. Unable to decode public key")
		return false
	}

	pubKeyHash, err := core.NewUserPubKeyHash(pubKeyBytes)
	if err!= nil{
		logger.WithFields(logger.Fields{
			"content"	: goAddr,
			"pubKey" 	: pubKeyBytes,
			"error" 	: err,
		}).Debug("Smart Contract: VerifyPublicKey failed. Unable to hash public key")
		return false
	}

	return pubKeyHash.GenerateAddress().String() == goAddr
}
