package client

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"

	"crypto/elliptic"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
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

func (ws *Wallets) LoadWalletFromFile() error {
	fileContent, err := storage.GetFileConnection(WalletFile)
	if err != nil {
		ws.SaveWalletToFile()
		fileContent, err = storage.GetFileConnection(WalletFile)
	}
	var wallets Wallets
	gob.Register(elliptic.P256())
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

	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	storage.SaveToFile(WalletFile, content)

}
