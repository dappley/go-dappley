package main

import (
	"context"
	"fmt"
)

func helpCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("Command: cli ", "createAccount")
	fmt.Println("Usage Example: cli createAccount")
	for cmd, pars := range cmdFlagsMap {
		fmt.Println("-----------------------------------------------------------------")
		fmt.Println("Command: cli ", cmd)
		fmt.Printf("Usage Example: cli %s", cmd)
		for _, par := range pars {
			fmt.Printf(" -%s", par.name)
			if par.name == flagFromAddress {
				fmt.Printf(" dWRFRFyientRqAbAmo6bskp9sBCTyFHLqF ")
				continue
			}
			if par.name == flagData {
				fmt.Printf(" helloworld! ")
				continue
			}
			if par.name == flagStartBlockHashes {

				fmt.Printf(" 8334b4c19091ae7582506eec5b84bfeb4a5e101042e40b403490c4ceb33897ba, 8334b4c19091ae7582506eec5b84bfeb4a5e101042e40b403490c4ceb33897bb ")
				continue
			}
			if par.name == flagPeerFullAddr {
				fmt.Printf(" /ip4/127.0.0.1/tcp/12345/ipfs/QmT5oB6xHSunc64Aojoxa6zg9uH31ajiAVyNfCdBZiwFTV ")
				continue
			}
			switch par.valueType {
			case valueTypeInt:
				fmt.Printf(" 10 ")
			case valueTypeString:
				fmt.Printf(" 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7 ")
			case valueTypeUint64:
				fmt.Printf(" 50 ")
			}

		}
		fmt.Println()
		fmt.Println("Arguments:")
		for _, par := range pars {
			fmt.Println(par.name, "\t", par.usage)
		}
		fmt.Println()
	}
}
