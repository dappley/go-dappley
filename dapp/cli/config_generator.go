package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

func configGeneratorCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {

	node := new(Node)

	//Check and set the node type
	for {
		fmt.Println("Choose config type (FullNode or MinerNode):")
		fmt.Scanln(&node.nodeType)
		setNodeType(node, strings.ToLower(node.nodeType))

		if node.NodeType == MinerNode || node.NodeType == FullNode {
			break
		} else {
			fmt.Println("Invalid input. To choose node type, input FullNode or MinerNode")
		}
	}

	//Name file
	fmt.Println("File name, don't include extension: (Input nothing for default: \"node\")")

	fmt.Scanln(&node.fileName)
	if node.fileName == "" {
		node.fileName = "node"
	}

	if node.NodeType == MinerNode {
		fmt.Println("Miner address info: (Input nothing to generate new key pair or input valueable key pair)")
		for {
			fmt.Print("miner_address: ")
			fmt.Scanln(&node.miner_address)
			if node.miner_address == "" {
				acc := createAccount(ctx, c, flags)
				if acc == nil {
					continue
				}
				node.miner_address = acc.GetAddress().String()
				pvk := acc.GetKeyPair().GetPrivateKey()
				hex_private_key, err1 := secp256k1.FromECDSAPrivateKey(&pvk)
				if err1 != nil {
					//err = err1
					return
				}
				node.private_key = hex.EncodeToString(hex_private_key)
				fmt.Println("New miner_address & private_key generated")
				fmt.Println("miner_address: ", node.miner_address)
				fmt.Println("private_key: ", node.private_key)
				break
			} else {
				fmt.Print("private_key: ")
				fmt.Scanln(&node.private_key)
				fmt.Println("Verifying account information....")
				acc := account.NewAccountByPrivateKey(node.private_key)
				if acc.GetAddress().String() == node.miner_address {
					break
				} else {
					fmt.Println("miner_address and private_key doesn't match")
				}
			}
		}
		node.node_address = node.miner_address
	}

	for {
		fmt.Println("Port info: (Input nothing for default setting: 12341)")
		fmt.Scanln(&node.port)
		if node.port == "" {
			node.port = "12341"
			break
		} else if _, err := strconv.Atoi(node.port); err != nil {
			fmt.Println("Input must be integer")
		} else {
			break
		}
	}

	for {

		fmt.Println("Seed: ")

		fmt.Scanln(&node.seed)
		if len(node.seed) <= 32 {
			fmt.Println("Please input a valid seed")
		} else {
			break
		}
	}
	fmt.Println("db_path: (Input nothing for default: ../bin/" + node.fileName + ".db)")

	fmt.Scanln(&node.db_path)
	if node.db_path == "" {
		node.db_path = "../bin/"
	}

	for {
		fmt.Println("Rpc_port: (Input nothing for default setting : 50051)")

		fmt.Scanln(&node.rpc_port)
		if node.rpc_port == "" {
			node.rpc_port = "50051"
			break
		} else if _, err := strconv.Atoi(node.rpc_port); err != nil {
			fmt.Println("Input must be integer")
		} else {
			break
		}
	}

	fmt.Println("Key: (Input nothing to generate new key)")
	fmt.Scanln(&node.key)
	if strings.ToLower(node.key) == "" {
		//generate key
		KeyPair, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)

		if err != nil {
			fmt.Printf("Generate key error %v\n", err)
			return
		}

		bytes, err := crypto.MarshalPrivateKey(KeyPair)
		if err != nil {
			fmt.Printf("MarshalPrivateKey error %v\n", err)
			return
		}
		str := base64.StdEncoding.EncodeToString(bytes)
		node.key = str
		fmt.Println("Key: " + node.key)
	}

	//write file name
	f, err := os.Create("../conf/" + node.fileName + ".conf")

	if err != nil {
		fmt.Println(err)
		return
	}

	defer f.Close()

	val := configContent(node)

	data := []byte(val)
	_, err = f.Write(data)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(node.fileName + ".conf" + " is created successfully")
	fmt.Println("Location: ../conf/" + node.fileName + ".conf")
}

func setNodeType(node *Node, nodeType string) {
	if nodeType == "minernode" {
		node.NodeType = MinerNode
	} else if nodeType == "fullnode" {
		node.NodeType = FullNode
	} else {
		node.NodeType = InvalidNode
	}
}

func configContent(node *Node) string {
	val1 := ("consensus_config{\n" +
		"	miner_address: " + "\"" + node.miner_address + "\"" + "\n" +
		"	private_key: \"" + node.private_key + "\"\n" +
		"}\n\n")
	val2 := ("node_config{\n" +
		"	port:	" + node.port + "\n" +
		"	seed:	[\"" + node.seed + "\"]\n" +
		"	db_path: \"" + node.db_path + node.fileName + ".db\"\n" +
		"	rpc_port: " + node.rpc_port + "\n")
	val3 := ("	key: \"" + node.key + "\"\n")
	val4 := ("	tx_pool_limit: 102400\n" +
		"	blk_size_limit: 102400\n" +
		"	node_address: \"" + node.node_address + "\"\n" +
		"	metrics_interval: 7200\n" +
		"	metrics_polling_interval: 5\n}")
	if node.NodeType == MinerNode && node.key == "" {
		val := val1 + val2 + val4
		return val
	} else if node.NodeType == MinerNode && node.key != "" {
		val := val1 + val2 + val3 + val4
		return val
	} else if node.NodeType == FullNode && node.key == "" {
		val := val2 + val4
		return val
	} else {
		val := val2 + val3 + val4
		return val
	}

}
