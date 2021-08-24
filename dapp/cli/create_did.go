package main

import (
	"context"
	"fmt"

	acc "github.com/dappley/go-dappley/core/account"
)

func createDIDCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	did := acc.NewDID()
	fmt.Println("Placeholder did is", did)
}
