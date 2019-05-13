package vm

/*
#include "v8/engine.h"
*/
import "C"
import "unsafe"

//export DeleteContract
func DeleteContract(address unsafe.Pointer) int {
	println("delete")
	return 0
}
