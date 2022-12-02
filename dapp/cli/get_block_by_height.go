package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dappley/go-dappley/common"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getBlockByHeightCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	blkHeight := uint64(*(flags[flagBlockHeight].(*int)))
	if blkHeight <= 0 {
		fmt.Println("\n Please enter a valid height. Example: cli getBlocksByHeight -height 5")
		fmt.Println()
		return
	}

	getBlockByHeightRequest := &rpcpb.GetBlockByHeightRequest{Height: blkHeight}

	response, err := c.(rpcpb.RpcServiceClient).RpcGetBlockByHeight(ctx, getBlockByHeightRequest)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}

	block := response.Block
	var encodedTransactions []map[string]interface{}
	for _, transaction := range block.GetTransactions() {
		var encodedVin []map[string]interface{}
		for _, vin := range transaction.GetVin() {
			encodedVin = append(encodedVin, map[string]interface{}{
				"Txid":      hex.EncodeToString(vin.GetTxid()),
				"Vout":      vin.GetVout(),
				"Signature": hex.EncodeToString(vin.GetSignature()),
				"PubKey":    hex.EncodeToString(vin.GetPublicKey()),
			})
		}

		var encodedVout []map[string]interface{}
		for _, vout := range transaction.GetVout() {
			encodedVout = append(encodedVout, map[string]interface{}{
				"Value":      common.NewAmountFromBytes(vout.GetValue()),
				"PubKeyHash": hex.EncodeToString(vout.GetPublicKeyHash()),
				"Contract":   vout.GetContract(),
			})
		}

		encodedTransaction := map[string]interface{}{
			"ID":   hex.EncodeToString(transaction.GetId()),
			"Vin":  encodedVin,
			"Vout": encodedVout,
		}
		encodedTransactions = append(encodedTransactions, encodedTransaction)
	}

	encodedBlock := map[string]interface{}{
		"Header": map[string]interface{}{
			"Hash":      hex.EncodeToString(block.GetHeader().GetHash()),
			"Prevhash":  hex.EncodeToString(block.GetHeader().GetPreviousHash()),
			"Timestamp": time.Unix(block.GetHeader().GetTimestamp(), 0).String(),
			"Sign":      hex.EncodeToString(block.GetHeader().GetSignature()),
			"height":    block.GetHeader().GetHeight(),
		},
		"Transactions": encodedTransactions,
	}

	blockJSON, err := json.MarshalIndent(encodedBlock, "", "  ")
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}

	fmt.Println(string(blockJSON))
}
