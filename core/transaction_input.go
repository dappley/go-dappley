package core

import(
	"bytes"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/core/pb"
)
type TXInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
}

// UsesKey checks whether the address initiated the transaction
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}


func (in *TXInput) ToProto() (proto.Message){
	return &corepb.TXInput{
		Txid:		in.Txid,
		Vout:		int32(in.Vout),
		Signature:	in.Signature,
		PubKey:		in.PubKey,
	}
}

func (in *TXInput) FromProto(pb proto.Message){
	in.Txid = pb.(*corepb.TXInput).Txid
	in.Vout = int(pb.(*corepb.TXInput).Vout)
	in.Signature = pb.(*corepb.TXInput).Signature
	in.PubKey = pb.(*corepb.TXInput).PubKey
}