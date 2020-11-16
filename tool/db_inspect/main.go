package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/utxo"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	db_inspect_pb "github.com/dappley/go-dappley/tool/db_inspect/pb"
	"github.com/dappley/go-dappley/util"
	"github.com/gogo/protobuf/proto"
)

type HandleFunc func()

type CommandInfo struct {
	command     string
	description string
	handle      HandleFunc
}

var commands = []CommandInfo{
	{"getBlock", "Get block information by hash", GetBlockHandle},
	{"getBlockByHeight", "Get block information by height", GetBlockByHeightHandle},
	{"getTransaction", "Get transaction information by hash", GetTransactionHandle},
	{"getCostTransaction", "Get cost transaction block", GetCostTransactionHandle},
	{"getUtxo", "Get utxo of address", GetUtxoHandle},
}

func GetBlockHandle() {
	var dbPath string
	var blockHash string

	flagSet := flag.NewFlagSet("getBlock", flag.ExitOnError)
	flagSet.StringVar(&dbPath, "d", "", "database path")
	flagSet.StringVar(&blockHash, "hash", "", "blk hash")
	flagSet.Parse(os.Args[2:])

	fmt.Printf("path: %v\n", dbPath)

	db := storage.OpenDatabase(dbPath)
	defer db.Close()

	blockHashBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		panic("Decode blk hash failed " + blockHash)
	}

	rawBytes, err := db.Get(blockHashBytes)
	if err != nil {
		panic("Block hash not found in database " + blockHash)
	}

	blk := block.Deserialize(rawBytes)
	if blk == nil {
		panic("Block deserialize failed with hash " + blockHash)
	}

	dumpBlock(blk)
}

func GetBlockByHeightHandle() {
	var dbPath string
	var blockHeight uint64

	flagSet := flag.NewFlagSet("getBlockByHeight", flag.ExitOnError)
	flagSet.StringVar(&dbPath, "d", "", "database path")
	flagSet.Uint64Var(&blockHeight, "height", 0, "blk height")
	flagSet.Parse(os.Args[2:])

	fmt.Printf("path: %v\n", dbPath)

	db := storage.OpenDatabase(dbPath)
	defer db.Close()

	hash, err := db.Get(util.UintToHex(blockHeight))
	if err != nil {
		panic(fmt.Sprintf("Block height %v not found in database ", blockHeight))
	}

	rawBytes, err := db.Get(hash)
	if err != nil {
		panic("Block hash not found in database " + hex.EncodeToString(hash))
	}

	blk := block.Deserialize(rawBytes)
	if blk == nil {
		panic("Block deserialize failed with hash " + hex.EncodeToString(hash))
	}

	dumpBlock(blk)
}

func GetTransactionHandle() {
	var dbPath string
	var txId string
	var startHeight uint64
	flagSet := flag.NewFlagSet("getTransaction", flag.ExitOnError)
	flagSet.StringVar(&dbPath, "d", "", "database path")
	flagSet.StringVar(&txId, "t", "", "transaction id")
	flagSet.Uint64Var(&startHeight, "height", 0, "search start block height")
	flagSet.Parse(os.Args[2:])

	db := storage.OpenDatabase(dbPath)
	defer db.Close()

	txIdBytes, err := hex.DecodeString(txId)
	if err != nil {
		panic("Decode transaction id failed " + txId)
	}

	for {
		hash, err := db.Get(util.UintToHex(startHeight))
		if err != nil {
			panic(fmt.Sprintf("Block height %v not found in database ", startHeight))
		}

		rawBytes, err := db.Get(hash)
		if err != nil {
			panic(fmt.Sprintf("Block hash %v not found in database ", hex.EncodeToString(hash)))
		}

		blk := block.Deserialize(rawBytes)
		if blk == nil {
			panic("Block deserialize failed with hash " + hex.EncodeToString(hash))
		}

		for _, tx := range blk.GetTransactions() {
			if bytes.Compare(txIdBytes, tx.ID) == 0 {
				dumpBlock(blk)
				return
			}
		}

		startHeight++
	}
}

func GetCostTransactionHandle() {
	var dbPath string
	var txId string
	var vOutIndex int
	var startHeight uint64
	flagSet := flag.NewFlagSet("getCostTransaction", flag.ExitOnError)
	flagSet.StringVar(&dbPath, "d", "", "database path")
	flagSet.StringVar(&txId, "t", "", "transaction id")
	flagSet.IntVar(&vOutIndex, "i", 0, "Vout index")
	flagSet.Uint64Var(&startHeight, "height", 0, "search start block height")
	flagSet.Parse(os.Args[2:])

	db := storage.OpenDatabase(dbPath)
	defer db.Close()

	txIdBytes, err := hex.DecodeString(txId)
	if err != nil {
		panic("Decode transaction id failed " + txId)
	}

	for {
		hash, err := db.Get(util.UintToHex(startHeight))
		if err != nil {
			panic(fmt.Sprintf("Block height %v not found in database ", startHeight))
		}

		rawBytes, err := db.Get(hash)
		if err != nil {
			panic(fmt.Sprintf("Block hash %v not found in database ", hex.EncodeToString(hash)))
		}

		blk := block.Deserialize(rawBytes)
		if blk == nil {
			panic("Block deserialize failed with hash " + hex.EncodeToString(hash))
		}

		for _, tx := range blk.GetTransactions() {
			for _, vin := range tx.Vin {
				if bytes.Compare(txIdBytes, vin.Txid) == 0 && vOutIndex == vin.Vout {
					dumpBlock(blk)
					return
				}
			}
		}

		startHeight++
	}
}

func GetUtxoHandle() {
	var dbPath string
	var address string

	flagSet := flag.NewFlagSet("getUtxo", flag.ExitOnError)
	flagSet.StringVar(&dbPath, "d", "", "database path")
	flagSet.StringVar(&address, "a", "", "search utxo address")
	flagSet.Parse(os.Args[2:])

	addr := account.NewAddress(address)
	acc := account.NewTransactionAccountByAddress(addr)
	if !acc.IsValid() {
		panic("Decode address failed")
	}
	pubKeyHash := acc.GetPubKeyHash()
	db := storage.OpenDatabase(dbPath)
	defer db.Close()

	utxoCache := utxo.NewUTXOCache(db)
	utxoTx := utxoCache.GetUTXOTx(pubKeyHash)

	dumpUtxos(utxoTx)
}

func block2PrettyPb(block *block.Block) proto.Message {
	blockHeaderPb := &db_inspect_pb.BlockHeader{
		Hash:         block.GetHash().String(),
		PreviousHash: block.GetPrevHash().String(),
		Nonce:        block.GetNonce(),
		Timestamp:    block.GetTimestamp(),
		Signature:    block.GetSign().String(),
		Height:       block.GetHeight(),
	}

	var txArray []*db_inspect_pb.Transaction
	for _, tx := range block.GetTransactions() {
		var txVinPbs []*db_inspect_pb.TXInput
		var txVoutPbs []*db_inspect_pb.TXOutput

		for _, vin := range tx.Vin {
			txVinPbs = append(txVinPbs,
				&db_inspect_pb.TXInput{
					Txid:      hex.EncodeToString(vin.Txid),
					Vout:      int32(vin.Vout),
					Signature: hex.EncodeToString(vin.Signature),
					PublicKey: hex.EncodeToString(vin.PubKey),
				})
		}

		for _, vout := range tx.Vout {
			txVoutPbs = append(txVoutPbs,
				&db_inspect_pb.TXOutput{
					Value:         vout.Value.String(),
					PublicKeyHash: vout.PubKeyHash.String(),
					Contract:      vout.Contract,
				})
		}

		txArray = append(
			txArray,
			&db_inspect_pb.Transaction{
				Id:       hex.EncodeToString(tx.ID),
				Vin:      txVinPbs,
				Vout:     txVoutPbs,
				Tip:      tx.Tip.String(),
				GasLimit: tx.GasLimit.String(),
				GasPrice: tx.GasPrice.String(),
			})
	}

	return &db_inspect_pb.Block{
		Header:       blockHeaderPb,
		Transactions: txArray,
	}
}

func dumpBlock(block *block.Block) {
	fmt.Print(proto.MarshalTextString(block2PrettyPb(block)))
}

func utxo2PrettyPb(utxo *utxo.UTXO) proto.Message {
	return &db_inspect_pb.Utxo{
		Amount:        utxo.Value.String(),
		PublicKeyHash: utxo.PubKeyHash.String(),
		Txid:          hex.EncodeToString(utxo.Txid),
		TxIndex:       uint32(utxo.TxIndex),
	}
}

func dumpUtxos(utxos *utxo.UTXOTx) {
	for _, utxo := range utxos.Indices {
		fmt.Print(proto.MarshalTextString(utxo2PrettyPb(utxo)))
	}
}

func printUsage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("\tdb_inspect <commands> [options]\n\n")
	fmt.Printf("The commands are:\n\n")

	for _, command := range commands {
		fmt.Printf("\t %v \t %v\n", command.command, command.description)
	}

	fmt.Printf("Use \"db_inspect <command> -h\" for more information about a command.\n")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	commandStr := os.Args[1]
	for _, command := range commands {
		if command.command == commandStr {
			command.handle()
			return
		}
	}

	printUsage()
}
