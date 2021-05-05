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
package transaction

import (
	"errors"

	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/transactionbase"
	transactionbasepb "github.com/dappley/go-dappley/core/transactionbase/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

var (
	ErrVoutNotFound = errors.New("vout not found in current transaction")
)

// TxJournal refers to transaction log data.
// It holds output array in each transactionbase.
type TxJournal struct {
	Txid []byte
	Vout []transactionbase.TXOutput
}

// Constructor
func NewTxJournal(txid []byte, vouts []transactionbase.TXOutput) *TxJournal {
	txJournal := &TxJournal{txid, vouts}
	return txJournal
}

// generate storage key in database
func getStorageKey(txid []byte) []byte {
	key := "tx_journal_" + string(txid)
	return []byte(key)
}

// Add new log
func PutTxJournal(tx Transaction, db storage.Storage) error {
	txJournal := NewTxJournal(tx.ID, tx.Vout)
	return txJournal.Save(db)
}

// Returns transaction log data from database
func GetTxOutput(vin transactionbase.TXInput, db storage.Storage) (transactionbase.TXOutput, error) {
	logger.Info("in Revert GetTxOutput")

	key := getStorageKey(vin.Txid)
	value, err := db.Get(key)
	if err != nil {
		logger.Warn(err,"vin.txid:",vin.Txid,",vin.Vout:",vin.Vout,"vin.PubKey:",vin.PubKey,",vin.Signature:",vin.Signature)
		return transactionbase.TXOutput{}, err
	}
	txJournal, err := DeserializeJournal(value)
	if err != nil {
		logger.Warn(err)
		return transactionbase.TXOutput{}, err
	}
	logger.Warn("vin.Vout: ",vin.Vout)
	if vin.Vout >= len(txJournal.Vout) {
		logger.Warn(ErrVoutNotFound)
		return transactionbase.TXOutput{}, ErrVoutNotFound
	}
	return txJournal.Vout[vin.Vout], nil
}

// Save TxJournal into database
func (txJournal *TxJournal) Save(db storage.Storage) error {
	bytes, err := txJournal.SerializeJournal()
	if err != nil {
		return err
	}
	err = db.Put(getStorageKey(txJournal.Txid), bytes)
	return err
}

func (txJournal *TxJournal) SerializeJournal() ([]byte, error) {
	rawBytes, err := proto.Marshal(txJournal.toProto())
	if err != nil {
		logger.WithError(err).Panic("TransactionJournal: Cannot serialize transactionJournal!")
		return nil, err
	}
	return rawBytes, nil
}

func DeserializeJournal(b []byte) (*TxJournal, error) {
	pb := &transactionpb.TransactionJournal{}
	err := proto.Unmarshal(b, pb)
	if err != nil {
		logger.WithError(err).Panic("TransactionJournal: Cannot deserialize transactionJournal!")
		return &TxJournal{}, err
	}
	txJournal := &TxJournal{}
	txJournal.fromProto(pb)
	return txJournal, nil
}

func (txJournal *TxJournal) toProto() proto.Message {
	var voutArray []*transactionbasepb.TXOutput
	for _, txout := range txJournal.Vout {
		voutArray = append(voutArray, txout.ToProto().(*transactionbasepb.TXOutput))
	}
	return &transactionpb.TransactionJournal{
		Vout: voutArray,
	}
}

func (txJournal *TxJournal) fromProto(pb proto.Message) {
	var voutArray []transactionbase.TXOutput
	txout := transactionbase.TXOutput{}
	for _, txoutpb := range pb.(*transactionpb.TransactionJournal).GetVout() {
		txout.FromProto(txoutpb)
		voutArray = append(voutArray, txout)
	}
	txJournal.Vout = voutArray
}
