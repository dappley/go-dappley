package main

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/wallet"
)

func listAddressesCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	if flags[flagListPrivateKey] == nil {
		fmt.Println("Error: privateKey is nil!")
		return
	}

	listPriv := *(flags[flagListPrivateKey].(*bool))

	passphrase := ""
	prompter := util.NewTerminalPrompter()

	empty, err := logic.IsAccountEmpty()
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	if empty {
		fmt.Println("Please use cli createAccount to generate an account first!")
		return
	}

	passphrase = prompter.GetPassPhrase("Please input the password: ", false)
	if passphrase == "" {
		fmt.Println("Password should not be empty!")
		return
	}
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	addressList, err := am.GetAddressesWithPassphrase(passphrase)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	if !listPriv {
		if len(addressList) == 0 {
			fmt.Println("The addresses in the account is empty!")
		} else {
			i := 1
			fmt.Println("The address list:")
			for _, addr := range addressList {
				fmt.Printf("Address[%d]: %s\n", i, addr)
				i++
			}
			fmt.Println()
			fmt.Println("Use the command 'cli listAddresses -privateKey' to list the addresses with private keys")
		}
	} else {
		privateKeyList := []string{}
		for _, addr := range addressList {
			keyPair := am.GetKeyPairByAddress(account.NewAddress(addr))
			pvk := keyPair.GetPrivateKey()
			privateKey, err := secp256k1.FromECDSAPrivateKey(&pvk)
			if err != nil {
				fmt.Println("Error: ", err.Error())
				return
			}
			privateKeyList = append(privateKeyList, hex.EncodeToString(privateKey))
		}
		if len(addressList) == 0 {
			fmt.Println("The addresses in the account is empty!")
		} else {
			i := 1
			fmt.Println("The address list with private keys:")
			for _, addr := range addressList {
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Printf("Address[%d]: %s \nPrivate Key[%d]: %s", i, addr, i, privateKeyList[i-1])
				fmt.Println()
				i++
			}
			fmt.Println("--------------------------------------------------------------------------------")
		}

	}

	return
}
