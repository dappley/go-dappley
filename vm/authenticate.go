package vm

import "C"
import (
	"encoding/hex"
	"encoding/pem"
	"crypto/ecdsa"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/crypto/x509"
	"github.com/spf13/viper"
	"io/ioutil"
	"unsafe"
)
import logger "github.com/sirupsen/logrus"

//export AuthenticateInitWithCertFunc
func AuthenticateInitWithCertFunc(address unsafe.Pointer, cert *C.char) bool {
	defer func() {
		if err := recover(); err != nil{
			logger.Errorf("catch error: %v", err)
		}
	}()

	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)
	goCert := C.GoString(cert)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
		}).Debug("SmartContract: failed to get state handler!")
		return false
	}

	rootCertPath := viper.GetString("auth.capath")
	logger.WithFields(logger.Fields{"auth.capath": rootCertPath}).Debug("ca file path")
	if rootCertPath == ""{
		rootCertPath = "conf/ca.crt"
	}

	rootPEMBlock, err := ioutil.ReadFile(rootCertPath)
	if (err != nil){
		logger.Errorf("read file ca.crt error: %v", err.Error())
		return false
	}

	rootBlock, _ := pem.Decode(rootPEMBlock)
	if rootBlock == nil{
		logger.Errorf("pem decode error")
		return  false
	}


	root, err := x509.ParseCertificate(rootBlock.Bytes)
	if err != nil{
		logger.Errorf("parse certificate root error :%v",err.Error())
	}

	roots := x509.NewCertPool()
	roots.AddCert(root)
	opts := x509.VerifyOptions{
		Roots:         roots,
	}

	block, _ := pem.Decode([]byte(goCert))
	subCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil{
		logger.Infof("Parse certificate failed, %v",err.Error())
		return false
	}

	_, err = subCert.Verify(opts)
	if err != nil{
		logger.Errorf("verify cert failed, error :%v", err.Error())
		return false
	}

	rsaPublicKey := subCert.PublicKey.(*ecdsa.PublicKey)
	pubKey, err := secp256k1.FromECDSAPublicKey(rsaPublicKey)
	if err != nil{
		logger.Infof("FromECDSAPublicKey failed, %v",err.Error())
		return false
	}

	strPubKey := hex.EncodeToString(pubKey[1:])
	logger.Infof("verify success , strPubKey: %v", strPubKey)
	engine.state.GetStorageByAddress(engine.contractAddr.String())[strPubKey] = goCert
	return true
}

//export AuthenticateVerifyWithPublicKeyFunc
func AuthenticateVerifyWithPublicKeyFunc(address unsafe.Pointer) bool {
	defer func() {
		if err := recover(); err != nil{
			logger.Errorf("catch error: %v", err)
		}
	}()

	addr := uint64(uintptr(address))
	engine := getV8EngineByAddress(addr)

	if engine == nil {
		logger.WithFields(logger.Fields{
			"contract_address": addr,
		}).Errorf("SmartContract: failed to get state handler!")
		return false
	}

	 if engine.tx == nil{
		 logger.WithFields(logger.Fields{
			 "contract_address": addr,
		 }).Errorf("SmartContract: failed to get transaction in v8 engine!")
	 }

	if len(engine.tx.Vin) == 0{
		logger.WithFields(logger.Fields{
			"contract_address": addr,
		}).Errorf("SmartContract: vin is empty!")
	}

	strPubKey := hex.EncodeToString(engine.tx.Vin[0].PubKey)

	if _,ok := engine.state.GetStorageByAddress(engine.contractAddr.String())[strPubKey];ok{
		return true
		logger.Infof("get pubkey success , strPubKey: %v", strPubKey)
	}

	logger.Infof("get pubkey failed , strPubKey: %v", strPubKey)
	return false
}











