package main

import (
	"context"
	"fmt"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
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
	if filepath == "" {
		for _, didSet := range dm.DIDSets {
			if didSet.DID == did {
				filepath = didSet.FileName
				fmt.Println("Found did document: ", filepath)
				break
			}
		}
	}
	if filepath == "" {
		fmt.Println("Could not find did.")
		return
	}

	doc, err := account.ReadDocFile(filepath)
	if err != nil {
		fmt.Println("Error reading DID document.")
		return
	}
	account.DisplayDIDDocument(*doc)
}
