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

package transactionbase

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	transactionbasepb "github.com/dappley/go-dappley/core/transactionbase/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type TXOutput struct {
	Value      *common.Amount
	PubKeyHash account.PubKeyHash
	Contract   string
}

func (out *TXOutput) GetAddress() account.Address {
	return out.PubKeyHash.GenerateAddress()
}

func NewTXOutput(value *common.Amount, account *account.TransactionAccount) *TXOutput {
	return NewTxOut(value, account, "")
}

func NewContractTXOutput(account *account.TransactionAccount, contract string) *TXOutput {
	return NewTxOut(common.NewAmount(0), account, contract)
}

func NewTxOut(value *common.Amount, account *account.TransactionAccount, contract string) *TXOutput {
	txo := &TXOutput{value, account.GetPubKeyHash(), contract}
	return txo
}

func (out *TXOutput) IsFoundInRewardStorage(rewardStorage map[string]string) bool {

	val, isFound := rewardStorage[out.GetAddress().String()]
	if !isFound {
		return false
	}

	amount, err := common.NewAmountFromString(val)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"reward": val,
		}).Warn("TXOutput: Reward amount is in invalid format.")
		return false
	}

	return out.Value.Cmp(amount) == 0
}

func (out *TXOutput) ToProto() proto.Message {
	return &transactionbasepb.TXOutput{
		Value:         out.Value.Bytes(),
		PublicKeyHash: []byte(out.PubKeyHash),
		Contract:      out.Contract,
	}
}

func (out *TXOutput) FromProto(pb proto.Message) {
	out.Value = common.NewAmountFromBytes(pb.(*transactionbasepb.TXOutput).GetValue())
	out.PubKeyHash = account.PubKeyHash(pb.(*transactionbasepb.TXOutput).GetPublicKeyHash())
	out.Contract = pb.(*transactionbasepb.TXOutput).GetContract()
}
