package account

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/dappley/go-dappley/crypto/hash"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

type DIDSet struct {
	PrivateKey ecdsa.PrivateKey
	FileName   string
	DID        string
}

type VerificationMethod struct {
	ID         string
	MethodType string
	Controller string
	Key        string
}

type BasicKey struct {
	ID         string
	MethodType string
	Key        string
}

type BasicDIDDocument struct {
	Context        string
	PublicKeys     []BasicKey
	Authentication []string
}

type FullDIDDocument struct {
	ID                  string
	VerificationMethods []VerificationMethod
	Context             string
	Authentication      []string
	OtherValues         map[string]string
}

const BasicFilePath = "../bin/basicDocs/"
const FullFilePath = "../bin/fullDocs/"

func CreateDIDDocument(name string) (*BasicDIDDocument, *DIDSet) {
	keys := NewKeyPair()
	didSet := &DIDSet{}
	didSet.PrivateKey = keys.GetPrivateKey()
	didSet.FileName = name
	basicKey := &BasicKey{}
	basicKey.ID = "#verification"
	basicKey.Key = hex.EncodeToString(keys.GetPublicKey())
	basicKey.MethodType = "Secp256k1"
	didDoc := &BasicDIDDocument{}
	didDoc.Context = "https://www.w3.org/ns/did/v1"
	didDoc.PublicKeys = append(didDoc.PublicKeys, *basicKey)
	didDoc.Authentication = append(didDoc.Authentication, basicKey.ID)
	SaveBasicDocFile(didDoc, name)

	bytesToHash, err := os.ReadFile(BasicFilePath + name)
	if err != nil {
		fmt.Println("error reading file: ", err)
		return nil, nil
	}
	hashed := hash.Sha3256(bytesToHash)
	hexstring := hex.EncodeToString(hashed)
	did := "did:dappley:" + hexstring
	didSet.DID = did

	fullDoc := createFullDocFile(didDoc, did)
	SaveFullDocFile(fullDoc)
	return didDoc, didSet
}

func createFullDocFile(basicDoc *BasicDIDDocument, did string) *FullDIDDocument {
	fullDoc := &FullDIDDocument{}
	fullDoc.ID = did
	fullDoc.Context = basicDoc.Context
	for _, pk := range basicDoc.PublicKeys {
		vm := VerificationMethod{}
		vm.ID = fullDoc.ID + pk.ID
		vm.Controller = fullDoc.ID
		vm.MethodType = pk.MethodType
		vm.Key = pk.Key
		fullDoc.VerificationMethods = append(fullDoc.VerificationMethods, vm)
	}
	for _, auth := range basicDoc.Authentication {
		fullAuth := fullDoc.ID + auth
		fullDoc.Authentication = append(fullDoc.Authentication, fullAuth)
	}
	fullDoc.OtherValues = make(map[string]string)
	return fullDoc
}

func SaveFullDocFile(fullDoc *FullDIDDocument) {
	rawBytes, err := proto.Marshal(fullDoc.ToProto())
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
	}
	os.Mkdir(FullFilePath, 0777)
	if err := os.WriteFile(FullFilePath+fullDoc.ID+".dat", rawBytes, 0666); err != nil {
		logger.Warn("Save ", fullDoc.ID+".dat", " failed.")
	}
}

func ReadFullDocFile(path string) (*FullDIDDocument, error) {
	protoBytes, err := os.ReadFile(path)
	if err != nil {
		logger.Warn(err.Error())
		return nil, err
	}

	doc := &accountpb.DIDDocFile{}
	err = proto.Unmarshal(protoBytes, doc)
	if err != nil {
		logger.Warn("proto.Unmarshal error: ", err.Error())
		return nil, err
	}
	newDoc := FullDIDDocument{}
	newDoc.FromProto(doc)
	return &newDoc, nil
}

func SaveBasicDocFile(doc *BasicDIDDocument, name string) {

	v2message := proto.MessageV2(doc.ToProto())

	jsonBytes, err := protojson.Marshal(v2message)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
	}
	os.Mkdir(BasicFilePath, 0777)
	if err := os.WriteFile(BasicFilePath+name, jsonBytes, 0666); err != nil {
		logger.Warn("Save ", name, " failed.")
	}
}

func ReadBasicDocFile(path string) (*BasicDIDDocument, error) {
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		logger.Warn(err.Error())
		return nil, err
	}

	doc := &accountpb.BasicDIDDocFile{}
	err = protojson.Unmarshal(jsonBytes, doc)
	if err != nil {
		logger.Warn("json.Unmarshal error: ", err.Error())
		return nil, err
	}
	newDoc := BasicDIDDocument{}
	newDoc.FromProto(doc)
	return &newDoc, nil
}

func DisplayFullDoc(doc *FullDIDDocument) {
	fmt.Println("{")
	fmt.Println("\t\"@context\": \"" + doc.Context + "\",")
	fmt.Println("\t\"id\": \"" + doc.ID + "\",")
	fmt.Println("\t\"verificationMethod\": [")
	skipComma := true
	for _, vm := range doc.VerificationMethods {
		if !skipComma {
			fmt.Println(",")
		}
		skipComma = false
		fmt.Println("\t\t{")
		fmt.Println("\t\t\t\"id\": \"" + vm.ID + "\",")
		fmt.Println("\t\t\t\"controller\": \"" + vm.Controller + "\",")
		fmt.Println("\t\t\t\"type\": \"" + vm.MethodType + "\",")
		fmt.Println("\t\t\t\"publicKeyHex\": \"" + vm.Key + "\"")
		fmt.Print("\t\t}")
	}
	fmt.Println("\n\t],")
	fmt.Print("\t\"authentication\": [")
	skipComma = true
	for _, auth := range doc.Authentication {
		if !skipComma {
			fmt.Print(", ")
		}
		skipComma = false
		fmt.Print("\"" + auth + "\"")
	}
	fmt.Println("]")

	for key, value := range doc.OtherValues {
		fmt.Print(",\n\t\"" + key + "\": \"" + value + "\"")
	}
	fmt.Println("\n}")

}

func PrepareSignature(privBytes []byte) ([]byte, []byte) {
	timeBytes, _ := time.Now().MarshalText()
	timeHash := sha256.Sum256(timeBytes)
	sig, _ := secp256k1.Sign(timeHash[:], privBytes)
	return sig, timeHash[:]
}

func VerifySignature(pubKey BasicKey, sig, timeHash []byte) (bool, error) {
	pubBytes, _ := hex.DecodeString(pubKey.Key)
	originPub := make([]byte, 1+len(pubBytes))
	originPub[0] = 4
	copy(originPub[1:], pubBytes)

	success, err := secp256k1.Verify(timeHash[:], sig, originPub)
	return success, err
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

func (d *BasicDIDDocument) ToProto() proto.Message {
	keys := []*accountpb.BasicKey{}
	for _, key := range d.PublicKeys {
		keys = append(keys, key.ToProto().(*accountpb.BasicKey))
	}
	return &accountpb.BasicDIDDocFile{
		Context:        d.Context,
		PublicKey:      keys,
		Authentication: d.Authentication,
	}
}

func (d *BasicDIDDocument) FromProto(pb proto.Message) {
	d.Context = pb.(*accountpb.BasicDIDDocFile).Context
	keys := []BasicKey{}

	for _, keypb := range pb.(*accountpb.BasicDIDDocFile).PublicKey {
		key := BasicKey{}
		key.FromProto(keypb)
		keys = append(keys, key)
	}
	d.PublicKeys = keys
	d.Authentication = pb.(*accountpb.BasicDIDDocFile).Authentication
}

func (d *FullDIDDocument) ToProto() proto.Message {
	methods := []*accountpb.VerificationMethod{}
	for _, method := range d.VerificationMethods {
		methods = append(methods, method.ToProto().(*accountpb.VerificationMethod))
	}
	return &accountpb.DIDDocFile{
		Context:            d.Context,
		Id:                 d.ID,
		VerificationMethod: methods,
		Authentication:     d.Authentication,
		OtherValues:        d.OtherValues,
	}
}

func (d *FullDIDDocument) FromProto(pb proto.Message) {
	d.Context = pb.(*accountpb.DIDDocFile).Context
	d.ID = pb.(*accountpb.DIDDocFile).Id
	methods := []VerificationMethod{}

	for _, methodpb := range pb.(*accountpb.DIDDocFile).VerificationMethod {
		vmethod := VerificationMethod{}
		vmethod.FromProto(methodpb)
		methods = append(methods, vmethod)
	}
	d.VerificationMethods = methods
	d.Authentication = pb.(*accountpb.DIDDocFile).Authentication
	d.OtherValues = pb.(*accountpb.DIDDocFile).OtherValues
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

func (b *BasicKey) ToProto() proto.Message {
	return &accountpb.BasicKey{
		Id:           b.ID,
		Type:         b.MethodType,
		PublicKeyHex: b.Key,
	}
}

func (b *BasicKey) FromProto(pb proto.Message) {
	b.ID = pb.(*accountpb.BasicKey).Id
	b.MethodType = pb.(*accountpb.BasicKey).Type
	b.Key = pb.(*accountpb.BasicKey).PublicKeyHex
}
