package contract

import "C"
import "fmt"

// VerifyAddressFunc verify address is valid
//export VerifyAddressFunc
func VerifyAddressFunc(address *C.char) int {
	fmt.Println("Go:VerifyAddressFunc")
	fmt.Println(C.GoString(address))
	return 2
}