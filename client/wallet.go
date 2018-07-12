package client

import "github.com/dappley/go-dappley/core"

type Wallet struct {
	Key       *core.KeyPair
	Addresses []core.Address
}

func NewWallet() Wallet {
	wallet := Wallet{}
	wallet.Key = core.NewKeyPair()
	wallet.Addresses = append(wallet.Addresses, wallet.Key.GenerateAddress())
	return wallet
}

func (w Wallet) GetAddress() core.Address {
	return w.Addresses[0]
}

func (w Wallet) GetKeyPair() *core.KeyPair {
	return w.Key
}

func (w Wallet) GetAddresses() []core.Address {
	return w.Addresses
}

func (w Wallet) ContainAddress(address core.Address) bool {
	for _, value := range w.Addresses {
		if value == address {
			return true
		}
	}
	return false
}
