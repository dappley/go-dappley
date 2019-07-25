package account_logic

import "github.com/dappley/go-dappley/client"

//isContract checks if an address is a Contract address
func IsContract(a client.Address) (bool, error) {
	pubKeyHash, ok := client.GeneratePubKeyHashByAddress(a)
	if !ok {
		return false, client.ErrInvalidAddress
	}
	return pubKeyHash.IsContract()
}
