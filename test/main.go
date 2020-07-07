package main

import (
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
)

func main()  {
	fmt.Println(string([]byte("300c0338c4b0d49edc66113e3584e04c6b907f9ded711d396d522aae6a79be1a")))
	account:= account.NewAccount()
	fmt.Println("key:",account.GetAddress())
	PrivateKey, err := secp256k1.ToECDSAPrivateKey([]byte("300c0338c4b0d49edc66113e3584e04c6b907f9ded711d396d522aae6a79be1a"))
	if err != nil{
		fmt.Println(err)
	}
	publicKey ,err := secp256k1.FromECDSAPublicKey(&PrivateKey.PublicKey)
	if err != nil{
		fmt.Println(err)
	}
	fmt.Println(len(publicKey))
	address := GenerateAddress(publicKey[1:])
	fmt.Println("address :",address)
}
//GenerateAddress generates an address  from a public key hash
func GenerateAddress(pub []byte) string {
	checksum := account.Checksum(pub)
	fullPayload := append(pub, checksum...)
	return base58.Encode(fullPayload)
}