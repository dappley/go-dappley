package sc

import "C"
import (
	"unsafe"

	"github.com/dappley/go-dappley/core"
)

//VerifyAddressFunc verify address is valid
//export VerifyAddressFunc
func VerifyAddressFunc(address *C.char) bool {
	addr := core.NewAddress(C.GoString(address))
	return addr.ValidateAddress()
}

//TransferFunc transfer amount to address
//export TransferFunc
func TransferFunc(handler unsafe.Pointer, to *C.char, amount *C.char, tip *C.char) int {

	return 0
}
