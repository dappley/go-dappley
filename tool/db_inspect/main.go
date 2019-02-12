package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/tool/db_inspect/pb"
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
	{"getTransaction", "Get transaction information by hash", GetTransactionHandle},
	{"getCostTransaction", "Get cost transaction block", GetCostTransactionHandle},
	{"getUtxo", "Get utxo of address", GetUtxoHandle},
}

func GetBlockHandle() {
	var dbPath string
	var blockHash string

	flagSet := flag.NewFlagSet("getBlock", flag.ExitOnError)
	flagSet.StringVar(&dbPath, "d", "", "database path")
	flagSet.StringVar(&blockHash, "hash", "", "search start block hash")
	flagSet.Parse(os.Args[2:])

	fmt.Printf("path: %v\n", dbPath)

	db := storage.OpenDatabase(dbPath)
	defer db.Close()

	blockHashBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		panic("Decode block hash failed " + blockHash)
	}

	rawBytes, err := db.Get(blockHashBytes)
	if err != nil {
		panic("Block hash not found in database " + blockHash)
	}

	block := core.Deserialize(rawBytes)
	if block == nil {
		panic("Block deserialize failed with hash " + blockHash)
	}

	dumpBlock(block)
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

		block := core.Deserialize(rawBytes)
		if block == nil {
			panic("Block deserialize failed with hash " + hex.EncodeToString(hash))
		}

		for _, tx := range block.GetTransactions() {
			if bytes.Compare(txIdBytes, tx.ID) == 0 {
				dumpBlock(block)
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

		block := core.Deserialize(rawBytes)
		if block == nil {
			panic("Block deserialize failed with hash " + hex.EncodeToString(hash))
		}

		for _, tx := range block.GetTransactions() {
			for _, vin := range tx.Vin {
				if bytes.Compare(txIdBytes, vin.Txid) == 0 && vOutIndex == vin.Vout {
					dumpBlock(block)
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

	addressBytes, err := hex.DecodeString(address)
	if err != nil {
		panic("Decode address failed")
	}

	db := storage.OpenDatabase(dbPath)
	defer db.Close()

	utxoBytes, err := db.Get([]byte("utxo"))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(utxoBytes) == 0 {
		panic("utxo load failed")
	}

	var utxos map[string][]*core.UTXO
	decoder := gob.NewDecoder(bytes.NewReader(utxoBytes))
	err = decoder.Decode(&utxos)
	if err != nil {
		panic("Decode utxo failed")
	}

	addressUtxos, ok := utxos[string(addressBytes)]
	if !ok {
		panic("Utxo not found")
	}

	dumpUtxos(addressUtxos)
}

func block2PrettyPb(block *core.Block) proto.Message {
	blockHeaderPb := &db_inspect_pb.BlockHeader{
		Hash:         hex.EncodeToString(block.GetHash()),
		PreviousHash: hex.EncodeToString(block.GetPrevHash()),
		Nonce:        block.GetNonce(),
		Timestamp:    block.GetTimestamp(),
		Signature:    hex.EncodeToString(block.GetSign()),
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
					PublicKeyHash: hex.EncodeToString(vout.PubKeyHash),
					Contract:      vout.Contract,
				})
		}

		txArray = append(
			txArray,
			&db_inspect_pb.Transaction{
				Id:   hex.EncodeToString(tx.ID),
				Vin:  txVinPbs,
				Vout: txVoutPbs,
				Tip:  tx.Tip.String(),
			})
	}

	return &db_inspect_pb.Block{
		Header:       blockHeaderPb,
		Transactions: txArray,
	}
}

func dumpBlock(block *core.Block) {
	fmt.Print(proto.MarshalTextString(block2PrettyPb(block)))
}

func utxo2PrettyPb(utxo *core.UTXO) proto.Message {
	return &db_inspect_pb.Utxo{
		Amount:        utxo.Value.String(),
		PublicKeyHash: hex.EncodeToString(utxo.PubKeyHash),
		Txid:          hex.EncodeToString(utxo.Txid),
		TxIndex:       uint32(utxo.TxIndex),
	}
}

func dumpUtxos(utxos []*core.UTXO) {
	for _, utxo := range utxos {
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
