package main

import (
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/core"
)

func main() {
	cli := CLI{}
	var db = storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()
	cli.Run(*db)
}
