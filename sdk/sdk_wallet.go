package sdk

import (
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"sync"
)

type DappSdkWallet struct {
	addrs     []core.Address
	balances  []uint64
	wm        *client.WalletManager
	sdk       *DappSdk
	utxoIndex *core.UTXOIndex
	mutex     *sync.Mutex
}

//NewDappleySdkWallet creates a new NewDappleySdkWallet instance that connects to a Dappley node with grpc port
func NewDappleySdkWallet(numOfWallets uint32, password string, sdk *DappSdk) *DappSdkWallet {

	dappSdkWallet := &DappSdkWallet{
		sdk:   sdk,
		mutex: &sync.Mutex{},
	}

	var err error

	dappSdkWallet.wm, err = logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.WithError(err).Error("DappSdkWallet: Cannot get wallet manager.")
		return nil
	}

	dappSdkWallet.addrs = dappSdkWallet.wm.GetAddresses()
	numOfExisitingWallets := len(dappSdkWallet.addrs)

	for i := numOfExisitingWallets; i < int(numOfWallets); i++ {
		_, err := logic.CreateWalletWithpassphrase(password)
		if err != nil {
			logger.WithError(err).Error("DappSdkWallet: Cannot create new wallet.")
			return nil
		}
		logger.WithFields(logger.Fields{
			"address": dappSdkWallet.addrs[i],
		}).Info("DappSdkWallet: Wallet is created")
	}

	dappSdkWallet.addrs = dappSdkWallet.wm.GetAddresses()
	dappSdkWallet.ClearBalances()

	return dappSdkWallet
}

func (sdkw *DappSdkWallet) GetAddrs() []core.Address { return sdkw.addrs }

func (sdkw *DappSdkWallet) GetBalances() []uint64 { return sdkw.balances }

func (sdkw *DappSdkWallet) GetWalletManager() *client.WalletManager { return sdkw.wm }

func (sdkw *DappSdkWallet) GetUtxoIndex() *core.UTXOIndex { return sdkw.utxoIndex }

func (sdkw *DappSdkWallet) ClearBalances() {
	sdkw.utxoIndex = core.NewUTXOIndex(core.NewUTXOCache(storage.NewRamStorage()))
	sdkw.balances = make([]uint64, len(sdkw.addrs))
}

//UpdateBalances updates all the balances of the addresses in the wallet
func (sdkw *DappSdkWallet) DisplayBalances() {
	for i, addr := range sdkw.GetAddrs() {
		logger.WithFields(logger.Fields{
			"address": addr.String(),
			"balance": sdkw.balances[i],
		}).Info("DappSdkWallet: Updating wallet balance...")
	}
}

//UpdateFromServer updates the balance and utxos of all addresses in the wallet from the server
func (sdkw *DappSdkWallet) UpdateFromServer() error {
	sdkw.mutex.Lock()
	defer sdkw.mutex.Unlock()

	logger.Info("DappSdkWallet: Updating from server")

	sdkw.ClearBalances()

	for i, addr := range sdkw.addrs {

		kp := sdkw.wm.GetKeyPairByAddress(addr)
		_, err := core.NewUserPubKeyHash(kp.PublicKey)
		if err != nil {
			return err
		}

		utxos, err := sdkw.sdk.GetUtxoByAddr(addr)
		if err != nil {
			return err
		}

		for _, utxoPb := range utxos {
			utxo := core.UTXO{}
			utxo.FromProto(utxoPb)
			sdkw.utxoIndex.AddUTXO(utxo.TXOutput, utxo.Txid, utxo.TxIndex)
			sdkw.AddToBalance(i, utxo.TXOutput.Value.Uint64())
		}
	}

	return nil
}

//AddToBalance adds the difference to the current balance
func (sdkw *DappSdkWallet) AddToBalance(index int, difference uint64) {
	sdkw.balances[index] += difference
}

//SubstractFromBalance substracts the difference from the current balance
func (sdkw *DappSdkWallet) SubstractFromBalance(index int, difference uint64) {
	sdkw.balances[index] -= difference
}
