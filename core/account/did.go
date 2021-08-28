package account

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type DIDSet struct {
	DID        string
	PrivateKey ecdsa.PrivateKey
}

type VerificationMethod struct {
	ID         string
	MethodType string
	Controller string
	Key        string
}

type DIDDocument struct {
	ID                    string
	VerificationMethods   []*VerificationMethod
	AuthenticationMethods []*VerificationMethod
}

func NewDID() *DIDSet {
	didSet := &DIDSet{}
	keys := NewKeyPair()
	didSet.PrivateKey = keys.GetPrivateKey()
	pubKeyHash := PubKeyHash(generatePubKeyHash(keys.GetPublicKey()))
	pubKeyHash = append([]byte{versionContract}, pubKeyHash...)
	address := pubKeyHash.GenerateAddress()
	didSet.DID = "did:dappley:" + address.address

	return didSet
}

func CreateDIDDocument(didSet *DIDSet) *DIDDocument {
	didDoc := &DIDDocument{}
	didDoc.ID = didSet.DID

	verMethod := CreateVerificationMethod(didSet)
	didDoc.VerificationMethods = append(didDoc.VerificationMethods, verMethod)
	didDoc.AuthenticationMethods = didDoc.VerificationMethods
	return didDoc
}

func CreateVerificationMethod(didSet *DIDSet) *VerificationMethod {
	verMethod := &VerificationMethod{}
	verMethod.ID = didSet.DID + "#verification"
	verMethod.MethodType = "placholder"
	verMethod.Controller = didSet.DID
	verMethod.Key = "placeholder"
	return verMethod
}

func GetDIDAddress(did string) Address {
	fields := strings.Split(did, ":")
	addressString := fields[2]
	return NewAddress(addressString)
}

func CheckDIDFormat(did string) bool {
	fields := strings.Split(did, ":")
	if len(fields) != 3 {
		fmt.Println("Error: incorrect number of fields in DID")
		return false
	}
	if fields[0] != "did" {
		fmt.Println("Error: DID missing 'did' prefix")
		return false
	}
	if fields[1] != "dappley" {
		fmt.Println("Error: DID is not a dappley DID")
		return false
	}
	return true
}

func (d *DIDSet) ToProto() proto.Message {
	rawBytes, err := secp256k1.FromECDSAPrivateKey(&d.PrivateKey)
	if err != nil {
		logger.Error("DIDSet: ToProto: Can not convert private key to bytes")
	}
	return &accountpb.DIDSet{
		DID:        d.DID,
		PrivateKey: rawBytes,
	}
}

func (d *DIDSet) FromProto(pb proto.Message) {
	d.DID = pb.(*accountpb.DIDSet).DID
	privKey, err := secp256k1.ToECDSAPrivateKey(pb.(*accountpb.DIDSet).PrivateKey)
	if err != nil {
		logger.Error("DIDSet: FromProto: Can not convert bytes to private key")
	}
	d.PrivateKey = *privKey
}
