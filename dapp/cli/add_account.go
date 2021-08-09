package main

import (
	"context"
	"fmt"

	acc "github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/wallet"
)

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
