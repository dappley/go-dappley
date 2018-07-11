package client

import "github.com/dappley/go-dappley/core"

type Wallet struct {
	Key       *core.KeyPair
	Addresses []string
}

func NewWallet() Wallet {
	return Wallet{}
}

func (w Wallet) GetAddress() []string {
	return w.Addresses
}
