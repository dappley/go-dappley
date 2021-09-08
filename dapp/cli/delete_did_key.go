package main

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/wallet"
)

func deleteDIDKeyCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	dm, err := logic.GetDIDManager(wallet.GetDIDFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	did := *(flags[flagDID].(*string))
	if did == "" {
		fmt.Println("Please provide a did value.")
		return
	}

	targetSet := &account.DIDSet{}

	for _, didSet := range dm.DIDSets {
		if didSet.DID == did {
			targetSet = didSet
			break
		}
	}
	key, _ := secp256k1.FromECDSAPrivateKey(&targetSet.PrivateKey)
	fmt.Println(hex.EncodeToString(key))
}
