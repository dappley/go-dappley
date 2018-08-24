// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package client

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1/bitelliptic"
)

const WalletFile = "../bin/wallets.dat"

type Wallets struct {
	Wallets []Wallet
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}

	err := wallets.LoadWalletFromFile()

	return &wallets, err
}

func (ws *Wallets) CreateWallet() Wallet {
	wallet := NewWallet()

	ws.Wallets = append(ws.Wallets, wallet)

	return wallet
}

func (ws *Wallets) DeleteWallet(key *core.KeyPair) error {
	for i, value := range ws.Wallets {
		if value.Key == key {
			ws.Wallets = append(ws.Wallets[:i], ws.Wallets[i+1:]...)
			return nil
		}
	}

	return errors.New("wallet is not exist")

}

func (ws *Wallets) DeleteWallets() error {
	if len(ws.Wallets) == 0 {
		return errors.New("no wallet yet")
	}
	ws.Wallets = ws.Wallets[:0]
	return nil
}

func (ws *Wallets) GetAddresses() []core.Address {
	var addresses []core.Address

	for _, address := range ws.Wallets {
		addresses = append(addresses, address.GetAddresses()...)
	}

	return addresses
}

func (ws Wallets) GetKeyPairByAddress(address core.Address) core.KeyPair {
	for _, value := range ws.Wallets {

		if value.ContainAddress(address) {
			return *value.Key
		}
	}
	return core.KeyPair{}

}

func (ws Wallets) GetWalletByAddress(address core.Address) Wallet {
	for _, wallet := range ws.Wallets {
		if wallet.ContainAddress(address) {
			return wallet
		}
	}
	return Wallet{}
}

func (ws *Wallets) LoadWalletFromFile() error {
	fileContent, err := storage.GetFileConnection(WalletFile)
	if err != nil {
		ws.SaveWalletToFile()
		fileContent, err = storage.GetFileConnection(WalletFile)
	}
	var wallets Wallets

	gob.Register(bitelliptic.S256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets

	return nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveWalletToFile() {
	var content bytes.Buffer

	gob.Register(bitelliptic.S256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	storage.SaveToFile(WalletFile, content)

}
