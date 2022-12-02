package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dappley/go-dappley/common"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getBlocksCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	maxCount := int32(*(flags[flagBlockMaxCount].(*int)))
	if maxCount <= 0 {
		fmt.Println("\n Please enter maxCount. Example: cli getBlocks -startBlockHashes 10 -maxCount 5")
		fmt.Println()
		return
	}

	getBlocksRequest := &rpcpb.GetBlocksRequest{MaxCount: maxCount}

	// set startBlockHashes of getBlocksRequest if specified in flag
	startBlockHashesString := string(*(flags[flagStartBlockHashes].(*string)))
	if len(startBlockHashesString) > 0 {
		var startBlockHashes [][]byte
		for _, startBlockHash := range strings.Split(startBlockHashesString, ",") {
			startBlockHashInByte, err := hex.DecodeString(startBlockHash)
			if err != nil {
				fmt.Println("Error: ", err.Error())
				return
			}
			startBlockHashes = append(startBlockHashes, startBlockHashInByte)
		}
		getBlocksRequest = &rpcpb.GetBlocksRequest{MaxCount: maxCount, StartBlockHashes: startBlockHashes}
	}

	response, err := account.(rpcpb.RpcServiceClient).RpcGetBlocks(ctx, getBlocksRequest)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}

	var encodedBlocks []map[string]interface{}
	for _, block := range response.Blocks {

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

		encodedBlocks = append(encodedBlocks, encodedBlock)
	}

	blocks, err := json.MarshalIndent(encodedBlocks, "", "  ")
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}

	fmt.Println(string(blocks))
}
