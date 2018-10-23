package contract

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ldappleyv8
#include <stdlib.h>
#include "v8/engine.h"
*/
import "C"
import "unsafe"

type ScEngine struct{
	source string
}

//NewScEngine generates a new ScEngine instance
func NewScEngine(source string) *ScEngine{
	return &ScEngine{
		source: source,
	}
}

func (sc *ScEngine) Execute(){
	cSource := C.CString(sc.source)
	defer C.free(unsafe.Pointer(cSource))

	C.executeV8Script(cSource)
}