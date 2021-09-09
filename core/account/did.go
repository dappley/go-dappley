package account

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"

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
}

const basicFilePath = "../bin/basicDocs/"
const fullFilePath = "../bin/fullDocs/"

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

	bytesToHash, err := os.ReadFile(basicFilePath + name)
	if err != nil {
		fmt.Println("error reading file: ", err)
		return nil, nil
	}
	hashed := hash.Sha3256(bytesToHash)
	hexstring := hex.EncodeToString(hashed)
	did := "did:dappley:" + hexstring
	didSet.DID = did

	fullDoc := createFullDIDDocument(didDoc, did)
	saveFullDIDDocument(fullDoc)
	return didDoc, didSet
}

func createFullDIDDocument(basicDoc *BasicDIDDocument, hash string) *FullDIDDocument {
	fullDoc := &FullDIDDocument{}
	fullDoc.ID = "did:dappley:" + hash
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
	return fullDoc
}

func saveFullDIDDocument(fullDoc *FullDIDDocument) {
	rawBytes, err := proto.Marshal(fullDoc.ToProto())
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
	}
	os.Mkdir(fullFilePath, 0777)
	if err := os.WriteFile(fullFilePath+fullDoc.ID+".dat", rawBytes, 0666); err != nil {
		logger.Warn("Save ", fullDoc.ID+".dat", " failed.")
	}
}

func SaveBasicDocFile(doc *BasicDIDDocument, name string) {

	v2message := proto.MessageV2(doc.ToProto())

	jsonBytes, err := protojson.Marshal(v2message)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
	}
	os.Mkdir(basicFilePath, 0777)
	if err := os.WriteFile(basicFilePath+name, jsonBytes, 0666); err != nil {
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
