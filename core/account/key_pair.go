// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package account

import (
	"crypto/ecdsa"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type KeyPair struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewKeyPair() *KeyPair {
	private, public := newKeyPair()
	return &KeyPair{private, public}
}

func (kp KeyPair) GenerateAddress() Address {
	pubKeyHash, _ := NewUserPubKeyHash(kp.PublicKey)
	return pubKeyHash.GenerateAddress()
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	private, err := secp256k1.NewECDSAPrivateKey()
	if err != nil {
		logger.Panic(err)
	}

	pubKey, _ := secp256k1.FromECDSAPublicKey(&private.PublicKey)
	//remove the uncompressed point at pubKey[0]
	return *private, pubKey[1:]
}
func GetKeyPairByString(privateKey string) *KeyPair {
	private, err := secp256k1.HexToECDSAPrivateKey(privateKey)
	if err != nil {
		logger.Panic(err)
	}

	pubKey, _ := secp256k1.FromECDSAPublicKey(&private.PublicKey)
	return &KeyPair{*private, pubKey[1:]}
}

func (kp *KeyPair) ToProto() proto.Message {
	rawBytes, err := secp256k1.FromECDSAPrivateKey(&kp.PrivateKey)
	if err != nil {
		logger.Error("Keypair: ToProto: Can not convert private key to bytes")
	}
	return &accountpb.KeyPair{
		PrivateKey: rawBytes,
		PublicKey:  kp.PublicKey,
	}
}

func (kp *KeyPair) FromProto(pb proto.Message) {
	privKey, err := secp256k1.ToECDSAPrivateKey(pb.(*accountpb.KeyPair).GetPrivateKey())
	if err != nil {
		logger.Error("Keypair: FromProto: Can not convert bytes to private key")
	}
	kp.PrivateKey = *privKey
	kp.PublicKey = pb.(*accountpb.KeyPair).GetPublicKey()
}
