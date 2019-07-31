package account_logic

import "github.com/dappley/go-dappley/core/account"

//isContract checks if an address is a Contract address
func IsContract(a account.Address) (bool, error) {
	pubKeyHash, ok := account.GeneratePubKeyHashByAddress(a)
	if !ok {
		return false, account.ErrInvalidAddress
	}
	return pubKeyHash.IsContract()
}
