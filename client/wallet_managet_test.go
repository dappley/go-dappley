package client

import (
	"testing"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/dappley/go-dappley/storage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
)

func TestWalletManager_LoadFromFileExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().ReadFromFile()

	wm := NewWalletManager(mockStorage)
	wm.LoadFromFile()

}

func TestWalletManager_LoadFromFileNotExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)

	gomock.InOrder(
		mockStorage.EXPECT().ReadFromFile().Return(nil, errors.New("err")),
		mockStorage.EXPECT().SaveToFile(gomock.Any()),
		mockStorage.EXPECT().ReadFromFile(),
	)

	wm := NewWalletManager(mockStorage)
	wm.LoadFromFile()
}

func TestWalletManager_SaveWalletToFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := storage_mock.NewMockFileStorage(mockCtrl)
	mockStorage.EXPECT().SaveToFile(gomock.Any())
	wm := NewWalletManager(mockStorage)
	wm.SaveWalletToFile()

}

func TestWalletManager_AddWallet(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.AddWallet(wallet)

	assert.Equal(t,wallet,wm.Wallets[0])
}

func TestWallet_GetAddresses(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	assert.Equal(t, wallet.GetAddresses(),wm.GetAddresses())
}

func TestWallet_GetAddressesNoWallet(t *testing.T) {
	wm := NewWalletManager(nil)
	assert.Equal(t,[]core.Address(nil),wm.GetAddresses())
}

func TestWalletManager_GetWalletByAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	assert.Equal(t, wallet, wm.GetWalletByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetWalletByUnfoundAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	assert.Nil(t, wm.GetWalletByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetWalletByAddressNilInput(t *testing.T) {
	wm := NewWalletManager(nil)
	assert.Nil(t, wm.GetWalletByAddress(core.Address{}))
}

func TestWalletManager_GetKeyPairByAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	wm.Wallets = append(wm.Wallets, wallet)
	assert.Equal(t, wallet.Key, wm.GetKeyPairByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetKeyPairByUnfoundAddress(t *testing.T) {
	wm := NewWalletManager(nil)
	wallet := NewWallet()
	assert.Nil(t, wm.GetKeyPairByAddress(wallet.GetAddress()))
}

func TestWalletManager_GetKeyPairByAddressNilInput(t *testing.T) {
	wm := NewWalletManager(nil)
	assert.Nil(t, wm.GetKeyPairByAddress(core.Address{}))
}