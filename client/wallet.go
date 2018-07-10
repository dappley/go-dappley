package client

import "github.com/dappley/go-dappley/core"

type Wallet struct {
	key       *core.KeyPair
	addresses []string
}

func NewWallet() Wallet {
	return Wallet{}
}

func (w Wallet) GetAddress() []string {
	return w.addresses
}
