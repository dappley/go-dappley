package main

import (
	"encoding/base64"
	"fmt"

	"github.com/libp2p/go-libp2p-crypto"
)

func main() {
	key, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)

	if err != nil {
		fmt.Printf("Generate key error %v\n", err)
		return
	}

	bytes, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		fmt.Printf("MarshalPrivateKey error %v\n", err)
		return
	}

	str := base64.StdEncoding.EncodeToString(bytes)
	fmt.Printf("%v\n", str)
}
