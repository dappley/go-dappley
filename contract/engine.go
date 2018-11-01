package sc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ldappleyv8
#include <stdlib.h>
#include "v8/engine.h"
bool VerifyAddressFunc_cgo(const char *address);
*/
import "C"
import (
	"unsafe"
	"sync"
)

var v8once = sync.Once{}

type V8Engine struct{
	source string
}

func InitializeV8Engine(){
	C.Initialize()
	C.InitializeBlockchain((C.VerifyAddressFunc)(unsafe.Pointer(C.VerifyAddressFunc_cgo)))
}

//NewV8Engine generates a new V8Engine instance
func NewV8Engine() *V8Engine {
	v8once.Do(func(){InitializeV8Engine()})
	return &V8Engine{
		source: "",
	}
}

func (sc *V8Engine) ImportSourceCode(source string){
	sc.source = source
}

func (sc *V8Engine) Execute(){
	cSource := C.CString(sc.source)
	defer C.free(unsafe.Pointer(cSource))
	var handler uint64
	handler=0
	C.executeV8Script(cSource, C.uintptr_t(handler))
}