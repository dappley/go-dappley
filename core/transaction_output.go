package core

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/core/pb"
)

type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

func (out *TXOutput) Lock(address []byte) {
	out.PubKeyHash = HashAddress(address)
}

func HashAddress(address []byte) []byte{
	pubKeyHash := util.Base58Decode(address)
	return pubKeyHash[1 : len(pubKeyHash)-4]
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(address))
	return txo
}

func (out *TXOutput) ToProto() (proto.Message){
	return &corepb.TXOutput{
		Value:		int32(out.Value),
		PubKeyHash:	out.PubKeyHash,
	}
}

func (out *TXOutput) FromProto(pb proto.Message){
	out.Value = int(pb.(*corepb.TXOutput).Value)
	out.PubKeyHash = pb.(*corepb.TXOutput).PubKeyHash
}