package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/hash"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/wallet"
)

func updateDIDCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	dm, err := logic.GetDIDManager(wallet.GetDIDFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	filepath := *(flags[flagFilePath].(*string))
	did := *(flags[flagDID].(*string))
	if did == "" && filepath == "" {
		fmt.Println("Please provide either a file path or a DID.")
		return
	} else if did != "" && filepath != "" {
		fmt.Println("Only provide one of the file path or the DID.")
		return
	}
	private := ecdsa.PrivateKey{}
	if did == "" {
		bytesToHash, err := os.ReadFile(account.BasicFilePath + filepath)
		if err != nil {
			fmt.Println("error reading file: ", err)
			return
		}
		hashed := hash.Sha3256(bytesToHash)
		hexstring := hex.EncodeToString(hashed)
		did = "did:dappley:" + hexstring
	}
	for _, didSet := range dm.DIDSets {
		if didSet.DID == did {
			filepath = didSet.FileName
			private = didSet.PrivateKey
			fmt.Println("Found did document: ", filepath)
			break
		}
	}
	if filepath == "" {
		fmt.Println("Could not find did.")
		return
	}

	doc, err := account.ReadBasicDocFile(account.BasicFilePath + filepath)
	if err != nil {
		fmt.Println("Error reading DID document.")
		return
	}
	authID := doc.Authentication[0]
	authVM := account.BasicVM{}
	for _, vm := range doc.VerificationMethod {
		if authID == vm.ID {
			authVM = vm
			break
		}
	}
	if authVM.ID == "" {
		fmt.Println("Failed to find correct key")
	} else {
		fmt.Println("Correct key is ", authVM.ID)
	}
	privBytes, err := secp256k1.FromECDSAPrivateKey(&private)
	if err != nil {
		fmt.Println("Private key error: ", err)
		return
	}

	sig, timeHash := account.PrepareSignature(privBytes)

	success, err := account.VerifySignature(authVM, sig, timeHash)
	if !success {
		fmt.Println("Failed to authenticate: ", err)
		return
	}
	fmt.Println("Authentication successful!")

	fullPathName := account.FullFilePath + did + ".dat"
	fullDoc, err := account.ReadFullDocFile(fullPathName)
	if err != nil {
		fmt.Println("Could not open document for " + did)
		return
	}
	fmt.Println("Opened document, id is " + fullDoc.ID)
	fmt.Println()
	if fullDoc.OtherValues == nil {
		fullDoc.OtherValues = make(map[string]string)
	}
	account.DisplayFullDoc(fullDoc)
	fmt.Println()

	prompter := util.NewTerminalPrompter()

	for {

		key, err := prompter.Prompt("Enter the key of the value you wish to add/modify (leave blank to exit): ")
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		if key == "" {
			break
		}
		if key == "@context" || key == "id" || key == "verificationMethod" || key == "authentication" {
			fmt.Println("That value cannot be modified.")
			continue
		}
		if value, found := fullDoc.OtherValues[key]; found {
			fmt.Println("Value of \"" + value + "\" found for key \"" + key + "\".")
			mode, err := prompter.Prompt("Type 'del' to delete or 'mod' to modify value (enter any other value to cancel): ")
			if err != nil {
				fmt.Println("Error: ", err)
				continue
			}
			switch mode {
			case "del":
				delete(fullDoc.OtherValues, key)
				fmt.Println("Key deleted.")
				continue

			case "mod":
				newVal, err := prompter.Prompt("Enter the new value (leave blank to cancel): ")
				if err != nil {
					fmt.Println("Error: ", err)
					continue
				}
				if newVal != "" {
					fullDoc.OtherValues[key] = newVal
					fmt.Println("Value updated to \"" + newVal + "\".")
				}
				continue
			default:
				continue
			}

		} else {
			fmt.Println("No value found for the key \"" + key + "\". Creating new key...")
			newVal, err := prompter.Prompt("Enter the new value (leave blank to cancel): ")
			if err != nil {
				fmt.Println("Error: ", err)
				continue
			}
			if newVal != "" {
				fullDoc.OtherValues[key] = newVal
				fmt.Println("Key created with value of \"" + newVal + "\".")
			}
			continue
		}
	}
	account.SaveFullDocFile(fullDoc)
	fmt.Println("Changes saved!")
}
