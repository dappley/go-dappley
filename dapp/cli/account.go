package main

import (
	"context"
	"fmt"

	acc "github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/wallet"
)

func createAccountCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	acc := createAccount(ctx, account, flags)
	if acc == nil {
		fmt.Print("Failed to create account.")
		return
	}

	fmt.Printf("Account is created. The address is %s \n", acc.GetAddress().String())
}

func addAccountCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	privKey := *(flags[flagKey].(*string))
	if privKey == "" {
		fmt.Println("Error: private key is missing!")
		return
	}
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	newAccount := acc.NewAccountByPrivateKey(privKey)
	duplicateAccount := am.GetAccountByAddress(newAccount.GetAddress())

	if duplicateAccount != nil {
		fmt.Println("Error: account already exists in wallet.")
		return
	}
	am.AddAccount(newAccount)
	am.SaveAccountToFile()
	fmt.Printf("Account added. The address is %s \n", newAccount.GetAddress().String())
}

func deleteAccountCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	privKey := *(flags[flagKey].(*string))
	addressString := *(flags[flagAddress].(*string))

	if privKey == "" && addressString == "" {
		fmt.Println("Error: no identifier (private key/address) provided!")
		return
	}

	if privKey != "" && addressString != "" {
		fmt.Println("Error: only provide one identifier (private key/address)")
		return
	}

	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	var address acc.Address
	if addressString == "" {
		address = acc.NewAccountByPrivateKey(privKey).GetAddress()
	} else {
		address = acc.NewAddress(addressString)
	}
	success := am.DeleteAccount(address)
	if success {
		am.SaveAccountToFile()
		fmt.Println("Account successfully deleted.")
	} else {
		fmt.Println("Error: could not find account in wallet.")
	}
}
