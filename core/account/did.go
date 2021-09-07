package account

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type DIDSet struct {
	DID        string
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
	FileName   string
}

type VerificationMethod struct {
	ID         string
	MethodType string
	Controller string
	Key        string
}

type DIDDocument struct {
	Name string
	ID   string
	//Values              map[string]string
	VerificationMethods []VerificationMethod
	Authentication      []string
}

func NewDID(name string) *DIDSet {
	didSet := &DIDSet{}
	keys := NewKeyPair()
	didSet.PrivateKey = keys.GetPrivateKey()
	didSet.PublicKey = keys.GetPublicKey()
	pubKeyHash := PubKeyHash(generatePubKeyHash(didSet.PublicKey))
	pubKeyHash = append([]byte{versionContract}, pubKeyHash...)
	address := pubKeyHash.GenerateAddress()
	didSet.DID = "did:dappley:" + address.address
	didSet.FileName = name

	return didSet
}

func CreateDIDDocument(didSet *DIDSet, name string) *DIDDocument {
	didDoc := &DIDDocument{}
	didDoc.Name = name
	didDoc.ID = didSet.DID

	verMethod := CreateVerificationMethod(didSet)
	didDoc.VerificationMethods = append(didDoc.VerificationMethods, *verMethod)
	didDoc.Authentication = append(didDoc.Authentication, verMethod.ID)
	SaveDocFile(didDoc)
	return didDoc
}

func DisplayDIDDocument(doc DIDDocument) {
	fmt.Println("{")
	fmt.Println("\"id\": \"" + doc.ID + "\"")
	fmt.Println("\"verificationMethod\": [")
	omitComma := true
	for _, method := range doc.VerificationMethods {
		if !omitComma {
			fmt.Println(",")
		}
		omitComma = false
		fmt.Print(method.ToString())
	}
	fmt.Println("\n]")
	fmt.Print("\"authentication\": [")
	omitComma = true
	for _, authMethod := range doc.Authentication {
		if !omitComma {
			fmt.Print(",")
		}
		omitComma = false
		fmt.Print("\"" + authMethod + "\"")
	}
	fmt.Println("]")
	fmt.Println("}")
}

func CreateVerificationMethod(didSet *DIDSet) *VerificationMethod {
	verMethod := &VerificationMethod{}
	verMethod.ID = "#verification"
	verMethod.MethodType = "Secp256k1"
	verMethod.Controller = didSet.DID
	verMethod.Key = hex.EncodeToString(didSet.PublicKey)
	return verMethod
}

func (verMethod *VerificationMethod) ToString() string {
	return "{\n\t\"id\": \"" + verMethod.ID + "\",\n\t\"type\": \"" + verMethod.MethodType + "\",\n\t\"controller\": \"" + verMethod.Controller + "\",\n\t\"publicKeyHex\": \"" + verMethod.Key + "\"\n}"
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

func SaveDocFile(doc *DIDDocument) {

	jsonBytes, err := json.Marshal(doc.ToProto())
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
	}
	if err := os.WriteFile(doc.Name, jsonBytes, 0666); err != nil {
		logger.Warn("Save ", doc.Name, " failed.")
		return
	}
}

func ReadDocFile(path string) (*DIDDocument, error) {
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		logger.Warn(err.Error())
		return nil, err
	}

	doc := &accountpb.DIDDocFile{}
	err = json.Unmarshal(jsonBytes, doc)
	if err != nil {
		logger.Warn("json.Unmarshal error: ", err.Error())
		return nil, err
	}
	newDoc := DIDDocument{}
	newDoc.FromProto(doc)
	return &newDoc, nil
}

func (d *DIDSet) ToProto() proto.Message {
	rawBytes, err := secp256k1.FromECDSAPrivateKey(&d.PrivateKey)
	if err != nil {
		logger.Error("DIDSet: ToProto: Can not convert private key to bytes")
	}
	return &accountpb.DIDSet{
		DID:        d.DID,
		PrivateKey: rawBytes,
		FilePath:   d.FileName,
	}
}

func (d *DIDSet) FromProto(pb proto.Message) {
	d.DID = pb.(*accountpb.DIDSet).DID
	privKey, err := secp256k1.ToECDSAPrivateKey(pb.(*accountpb.DIDSet).PrivateKey)
	if err != nil {
		logger.Error("DIDSet: FromProto: Can not convert bytes to private key")
	}
	d.PrivateKey = *privKey
	d.FileName = pb.(*accountpb.DIDSet).FilePath
}

func (d *DIDDocument) ToProto() proto.Message {
	methods := []*accountpb.VerificationMethod{}
	for _, method := range d.VerificationMethods {
		methods = append(methods, method.ToProto().(*accountpb.VerificationMethod))
	}
	return &accountpb.DIDDocFile{
		Id:                 d.ID,
		VerificationMethod: methods,
		Authentication:     d.Authentication,
	}
}

func (d *DIDDocument) FromProto(pb proto.Message) {
	d.ID = pb.(*accountpb.DIDDocFile).Id

	methods := []VerificationMethod{}

	for _, methodpb := range pb.(*accountpb.DIDDocFile).VerificationMethod {
		vmethod := VerificationMethod{}
		vmethod.FromProto(methodpb)
		methods = append(methods, vmethod)
	}
	d.VerificationMethods = methods
	d.Authentication = pb.(*accountpb.DIDDocFile).Authentication
}

func (v *VerificationMethod) ToProto() proto.Message {
	return &accountpb.VerificationMethod{
		Id:           v.ID,
		Type:         v.MethodType,
		Controller:   v.Controller,
		PublicKeyHex: v.Key,
	}
}

func (v *VerificationMethod) FromProto(pb proto.Message) {
	v.ID = pb.(*accountpb.VerificationMethod).Id
	v.MethodType = pb.(*accountpb.VerificationMethod).Type
	v.Controller = pb.(*accountpb.VerificationMethod).Controller
	v.Key = pb.(*accountpb.VerificationMethod).PublicKeyHex
}
