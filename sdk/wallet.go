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
	balances  map[core.Address]uint64
	wm        *client.WalletManager
	sdk       *DappSdk
	utxoIndex *core.UTXOIndex
	mutex     *sync.Mutex
}

//NewDappSdkWallet creates a new NewDappSdkWallet instance that connects to a Dappley node with grpc port
func NewDappSdkWallet(numOfWallets uint32, password string, sdk *DappSdk) *DappSdkWallet {

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

func (sdkw *DappSdkWallet) GetBalance(address core.Address) uint64 { return sdkw.balances[address] }

func (sdkw *DappSdkWallet) GetWalletManager() *client.WalletManager { return sdkw.wm }

func (sdkw *DappSdkWallet) GetUtxoIndex() *core.UTXOIndex { return sdkw.utxoIndex }

func (sdkw *DappSdkWallet) ClearBalances() {
	sdkw.utxoIndex = core.NewUTXOIndex(core.NewUTXOCache(storage.NewRamStorage()))
	sdkw.balances = make(map[core.Address]uint64)
}

//UpdateBalances updates all the balances of the addresses in the wallet
func (sdkw *DappSdkWallet) DisplayBalances() {
	for _, addr := range sdkw.GetAddrs() {
		logger.WithFields(logger.Fields{
			"address": addr.String(),
			"balance": sdkw.balances[addr],
		}).Info("DappSdkWallet: Updating wallet balance...")
	}
}

//Update updates the balance and utxos of all addresses in the wallet from the server
func (sdkw *DappSdkWallet) Update() error {
	sdkw.mutex.Lock()
	defer sdkw.mutex.Unlock()

	logger.Info("DappSdkWallet: Updating from server")

	sdkw.ClearBalances()

	for _, addr := range sdkw.addrs {

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
			sdkw.UpdateBalance(addr, sdkw.GetBalance(addr)+utxo.TXOutput.Value.Uint64())
		}
	}

	return nil
}

//AddToBalance adds the difference to the current balance
func (sdkw *DappSdkWallet) UpdateBalance(addr core.Address, amount uint64) {
	sdkw.balances[addr] = amount
}
