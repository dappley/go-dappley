package main

import (
	"context"
	"fmt"

	"github.com/dappley/go-dappley/core/account"
)

func updateDIDCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	filepath := *(flags[flagFilePath].(*string))

	doc, err := account.ReadDocFile(filepath)
	if err != nil {
		fmt.Println("Error reading DID document.")
		return
	}
	account.DisplayDIDDocument(*doc)
}
