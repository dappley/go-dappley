package main

import (
	"context"
	"fmt"
)

func createAccountCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	acc := createAccount(ctx, account, flags)
	if acc == nil {
		fmt.Print("Failed to create account.")
		return
	}

	fmt.Printf("Account is created. The address is %s \n", acc.GetAddress().String())
}
