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

package core

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"
)

// UTXO contains the meta info of an unspent TXOutput.
type UTXO struct {
	TXOutput
	Txid    []byte
	TxIndex int
}

// PrepareUTXOs returns a minimum subset of utxos valued at amount or more
// for smart contract, utxos[0] is expected to be the contract
func PrepareUTXOs(utxos []*UTXO, amount *common.Amount) ([]*UTXO, bool) {
	sum := common.NewAmount(0)

	if len(utxos) < 1 {
		return utxos, false
	}

	if isContract, _ := utxos[0].PubKeyHash.IsContract(); isContract {
		utxos = utxos[1:]
	}

	for i, u := range utxos {
		sum = sum.Add(u.Value)
		if sum.Cmp(amount) >= 0 {
			return utxos[:i+1], true
		}
	}
	return utxos, false

}

// newUTXO returns an UTXO instance constructed from a TXOutput.
func newUTXO(txout TXOutput, txid []byte, vout int) *UTXO {
	return &UTXO{txout, txid, vout}
}

func getTXOutputSpent(in TXInput, bc *Blockchain) (TXOutput, int, error) {
	tx, err := bc.FindTransaction(in.Txid)
	if err != nil {
		return TXOutput{}, 0, ErrTXInputInvalid
	}
	return tx.Vout[in.Vout], in.Vout, nil
}

func (utxo *UTXO) ToProto() proto.Message {
	return &corepb.Utxo{
		Amount:        utxo.Value.Bytes(),
		PublicKeyHash: []byte(utxo.PubKeyHash),
		Txid:          utxo.Txid,
		TxIndex:       uint32(utxo.TxIndex),
	}
}

func (utxo *UTXO) FromProto(pb proto.Message) {
	utxopb := pb.(*corepb.Utxo)
	utxo.Value = common.NewAmountFromBytes(utxopb.Amount)
	utxo.PubKeyHash = utxopb.PublicKeyHash
	utxo.Txid = utxopb.Txid
	utxo.TxIndex = int(utxopb.TxIndex)
}
