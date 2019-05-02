package sdk

import (
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	logger "github.com/sirupsen/logrus"
)

type DappSdkWallet struct {
	conn     *DappSdkConn
	addrs    []core.Address
	balances []uint64
	wm       *client.WalletManager
}

//NewDappleySdkWallet creates a new NewDappleySdkWallet instance that connects to a Dappley node with grpc port
func NewDappleySdkWallet(conn *DappSdkConn, numOfWallets int, password string) *DappSdkWallet {

	dappleySdkWallet := &DappSdkWallet{
		conn: conn,
	}

	var err error

	dappleySdkWallet.wm, err = logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.WithError(err).Error("Cannot get wallet manager.")
		return nil
	}

	dappleySdkWallet.addrs = dappleySdkWallet.wm.GetAddresses()
	numOfExisitingWallets := len(dappleySdkWallet.addrs)

	for i := numOfExisitingWallets; i < numOfWallets; i++ {
		_, err := logic.CreateWalletWithpassphrase(password)
		if err != nil {
			logger.WithError(err).Error("Cannot create new wallet.")
			return nil
		}
		logger.WithFields(logger.Fields{
			"address": dappleySdkWallet.addrs[i],
		}).Info("Wallet is created")
	}

	dappleySdkWallet.addrs = dappleySdkWallet.wm.GetAddresses()
	dappleySdkWallet.balances = make([]uint64, len(dappleySdkWallet.addrs))

	return dappleySdkWallet
}

func (sdkw *DappSdkWallet) GetAddrs() []core.Address {
	return sdkw.addrs
}

func (sdkw *DappSdkWallet) GetConn() *DappSdkConn {
	return sdkw.conn
}

func (sdkw *DappSdkWallet) GetBalances() []uint64 { return sdkw.balances }

func (sdkw *DappSdkWallet) GetWalletManager() *client.WalletManager { return sdkw.wm }

func (sdkw *DappSdkWallet) UpdateBalancesFromServer(blockchain *DappSdkBlockchain) {
	for i, addr := range sdkw.GetAddrs() {
		amount, err := blockchain.GetBalance(addr.String())
		balanceLogger := logger.WithFields(logger.Fields{
			"address": addr.String(),
			"amount":  amount,
			"record":  sdkw.balances[i],
		})
		if err != nil {
			balanceLogger.WithError(err).Warn("Failed to get wallet balance.")
		}
		balanceLogger.Info("Updating wallet balance...")
		sdkw.balances[i] = uint64(amount)
	}
}

func (sdkw *DappSdkWallet) AddToBalance(index int, difference uint64) {
	sdkw.balances[index] += difference
}

func (sdkw *DappSdkWallet) SubstractFromBalance(index int, difference uint64) {
	sdkw.balances[index] -= difference
}
