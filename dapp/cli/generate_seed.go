package main

import (
	"context"
	"encoding/base64"
	"fmt"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

func generateSeedCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {

	key, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)

	if err != nil {
		fmt.Println("Generate key error: ", err)
		return
	}

	bytes, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		fmt.Println("MarshalPrivateKey error: ", err)
		return
	}

	str := base64.StdEncoding.EncodeToString(bytes)
	fmt.Printf("%v\n", str)

}
