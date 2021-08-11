package main

import (
	"os"
	"fmt"
	"context"
	"strconv"
	"strings"
	"encoding/hex"
	"encoding/base64"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
)
​
func configGeneratorCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	node := new(Node)
​
	// Select node configuration file type
	for {
		fmt.Println("Select node type - FullNode or MinerNode:")
		fmt.Scanln(&node.nodeType)
		setNodeType(node, strings.ToLower(node.nodeType))
		if node.NodeType == MinerNode || node.NodeType == FullNode {
			break
		} else {
			fmt.Println("Error: Node type must be either \"FullNode\" or \"MinerNode\"!")
		}
	}
​
	// Input name of the configuration file
	fmt.Println("Input file name - Input nothing for default name \"new_node\":")
	fmt.Scanln(&node.fileName)
	if node.fileName == "" {
		node.fileName = "new_node"
	}
​
	// When the node type is MinerNode, create the "consensus_config" section
	if node.NodeType == MinerNode {
		fmt.Println("Miner address and keypair - Input nothing to generate new address and keypair:")
		for {
			fmt.Println("Input miner address:")
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
				fmt.Println("New miner address and private keypair has been generated!")
				fmt.Println("Miner address:", node.miner_address)
				fmt.Println("Private keypair:", node.private_key)
				break
			} else {
				fmt.Println("Input private keypair:")
				fmt.Scanln(&node.private_key)
				fmt.Println("Verifying account information....")
				acc := account.NewAccountByPrivateKey(node.private_key)
				if acc.GetAddress().String() == node.miner_address {
					break
				} else {
					fmt.Println("Error: Miner address and private keypair do not match!")
				}
			}
		}
		node.node_address = node.miner_address
	}
​
	// Input node port info
	for {
		fmt.Println("Input port info - Input nothing for default port \"12341\":")
		fmt.Scanln(&node.port)
		if node.port == "" {
			node.port = "12341"
			break
		} else if _, err := strconv.Atoi(node.port); err != nil {
			fmt.Println("Error: Input must be an integer value!")
		} else {
			break
		}
	}
​
	// Input node seed info
	for {
		fmt.Println("Input seed:")
		fmt.Scanln(&node.seed)
		if len(node.seed) <= 32 {
			fmt.Println("Error: Invalid seed!")
		} else {
			break
		}
	}
​
	// Input db_path info
	fmt.Println("Input database path - Input nothing for default path \"../bin/" + node.fileName + ".db\"")
	fmt.Scanln(&node.db_path)
	if node.db_path == "" {
		node.db_path = "../bin/"
	}
​
	// Input rpc_port info
	for {
		fmt.Println("Input rpc port - Input nothing for default port \"50051\"")
		fmt.Scanln(&node.rpc_port)
		if node.rpc_port == "" {
			node.rpc_port = "50051"
			break
		} else if _, err := strconv.Atoi(node.rpc_port); err != nil {
			fmt.Println("Error: Input must be integer!")
		} else {
			break
		}
	}
​
	// Input key info
	fmt.Println("Input key - Input nothing to generate a new key:")
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
		fmt.Println("Key:", node.key)
	}
​
	// Create file
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
​
	// Finish
	fmt.Println(node.fileName + ".conf is created successfully!")
	fmt.Println("Location: ../conf/" + node.fileName + ".conf")
}
​
// Set node type to either MinerNode or FullNode
func setNodeType(node *Node, nodeType string) {
	if nodeType == "minernode" {
		node.NodeType = MinerNode
	} else if nodeType == "fullnode" {
		node.NodeType = FullNode
	} else {
		node.NodeType = InvalidNode
	}
}
​
// Creates the node configuration file content following its syntax rules
func configContent(node *Node) string {
	var final_content string
	consensus_config := ("consensus_config{\n" +
		"	miner_address: " + "\"" + node.miner_address + "\"" + "\n" +
		"	private_key: \"" + node.private_key + "\"\n" +
		"}\n\n")
	node_config_1 := ("node_config{\n" +
		"	port:	" + node.port + "\n" +
		"	seed:	[\"" + node.seed + "\"]\n" +
		"	db_path: \"" + node.db_path + node.fileName + ".db\"\n" +
		"	rpc_port: " + node.rpc_port + "\n")
	node_config_2 := ("	key: \"" + node.key + "\"\n")
	node_config_3 := ("	tx_pool_limit: 102400\n" +
		"	blk_size_limit: 102400\n" +
		"	node_address: \"" + node.node_address + "\"\n" +
		"	metrics_interval: 7200\n" +
		"	metrics_polling_interval: 5\n}\n\n\n")
	if node.NodeType == MinerNode && node.key == "" {
		final_content = consensus_config + node_config_1 + node_config_3
	} else if node.NodeType == MinerNode && node.key != "" {
		final_content = consensus_config + node_config_1 + node_config_2 + node_config_3
	} else if node.NodeType == FullNode && node.key == "" {
		final_content = node_config_1 + node_config_3
	} else {
		final_content = node_config_1 + node_config_2 + node_config_3
	}
	return final_content
}