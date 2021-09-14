package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/wallet"
)

func createDIDCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	dm, err := logic.GetDIDManager(wallet.GetDIDFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	prompter := util.NewTerminalPrompter()

	name, err := prompter.Prompt("Enter the name to be used for the new basic DID document: ")
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	name += ".json"
	if _, err := os.Stat(name); err == nil {
		fmt.Println("Error: file already exists.")
		return
	}

	didDoc, didSet := account.CreateDIDDocument(name)
	if didDoc == nil {
		fmt.Println("Could not create file.")
		return
	}

	dm.AddDID(didSet)
	dm.SaveDIDsToFile()
	fmt.Println("Operation complete! " + name + " has been created!")
}
