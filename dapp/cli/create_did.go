package main

import (
	"context"
	"fmt"
	"os"

	acc "github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/wallet"
)

func createDIDCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	dm, err := logic.GetDIDManager(wallet.GetDIDFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	prompter := util.NewTerminalPrompter()

	name, err := prompter.Prompt("Enter the name to be used for the new DID document: ")
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	if _, err := os.Stat(name + ".txt"); err == nil {
		fmt.Println("Error: file already exists.")
		return
	}

	didSet := acc.NewDID(name)
	if !acc.CheckDIDFormat(didSet.DID) {
		fmt.Println("DID formatted incorrectly.")
		return
	}

	didDoc := acc.CreateDIDDocument(didSet, name)
	if didDoc == nil {
		fmt.Println("Could not create file.")
		return
	}
	fmt.Println("Document created. New did is", didSet.DID)

	dm.AddDID(didSet)
	dm.SaveDIDsToFile()
	fmt.Println("Operation complete!")
}
