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
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/cipher"
	Logger "github.com/sirupsen/logrus"
)

const Algorithm = 1 << 4

type Wallet struct {
	Key       *core.KeyPair
	Addresses []core.Address
	Passphrase []byte
}

func NewWallet() *Wallet {
	wallet := &Wallet{}
	wallet.Key = core.NewKeyPair()
	wallet.Addresses = append(wallet.Addresses, wallet.Key.GenerateAddress())
	return wallet
}

func NewWalletWithPassphrase(passphrase string) *Wallet {
	wallet := &Wallet{}
	wallet.Key = core.NewKeyPair()
	wallet.Addresses = append(wallet.Addresses, wallet.Key.GenerateAddress())
	cipher := cipher.NewCipher(uint8(Algorithm))
	pass := []byte(passphrase)
	addressData := []byte(wallet.Addresses[0].Address)

	passBytes,err := cipher.Encrypt(addressData, pass)
	if err != nil {
		Logger.Error("New Wallet: Encrypt Error!")
		return nil
	}
	wallet.Passphrase = passBytes

	return wallet
}

func (w *Wallet) GetAddress() core.Address {
	return w.Addresses[0]
}

func (w *Wallet) GetKeyPair() *core.KeyPair {
	return w.Key
}

func (w *Wallet) GetAddresses() []core.Address {
	return w.Addresses
}

func (w *Wallet) ContainAddress(address core.Address) bool {
	for _, value := range w.Addresses {
		if value == address {
			return true
		}
	}
	return false
}
