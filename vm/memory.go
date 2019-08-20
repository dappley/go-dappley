package vm

//#include <stdlib.h>
import "C"

import (
	"unsafe"
)

//export Malloc
func Malloc(size C.size_t) unsafe.Pointer {
	return C.malloc(size)
}

//export Free
func Free(address unsafe.Pointer) {
	C.free(address)
}
