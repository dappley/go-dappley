package contract

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ldappleyv8
#include <stdlib.h>
#include "v8/engine.h"
int VerifyAddressFunc_cgo(const char *address);
*/
import "C"
import (
	"unsafe"
	"sync"
)

var v8once = sync.Once{}

type ScEngine struct{
	source string
}

func InitializeV8Engine(){
	C.Initialize()
	C.InitializeBlockchain((C.VerifyAddressFunc)(unsafe.Pointer(C.VerifyAddressFunc_cgo)))
}

//NewScEngine generates a new ScEngine instance
func NewScEngine(source string) *ScEngine{
	v8once.Do(func(){InitializeV8Engine()})
	return &ScEngine{
		source: source,
	}
}

func (sc *ScEngine) Execute(){
	cSource := C.CString(sc.source)
	defer C.free(unsafe.Pointer(cSource))
	var handler uint64
	handler=0
	C.executeV8Script(cSource, C.uintptr_t(handler))
}