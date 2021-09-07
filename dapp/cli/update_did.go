package main

import (
	"context"
	"fmt"
	"io/ioutil"
)

func updateDIDCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	filepath := *(flags[flagFilePath].(*string))

	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("Failed to read the DID document!")
		return
	}
	fmt.Println(string(bytes))
}
