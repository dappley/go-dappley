package sc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ldappleyv8
#include <stdlib.h>
#include "v8/engine.h"
//blockchain
bool  Cgo_VerifyAddressFunc(const char *address);
//storage
char* Cgo_StorageGetFunc(const char *key);
int   Cgo_StorageSetFunc(const char *key, const char *value);
int   Cgo_StorageDelFunc(const char *key);
*/
import "C"
import (
	"fmt"
	"unsafe"
	"sync"
)

var v8once = sync.Once{}

type V8Engine struct{
	source string
}

func InitializeV8Engine(){
	C.Initialize()
	C.InitializeBlockchain((C.FuncVerifyAddress)(unsafe.Pointer(C.Cgo_VerifyAddressFunc)))
	C.InitializeStorage((C.FuncStorageGet)(unsafe.Pointer(C.Cgo_StorageGetFunc)),
						(C.FuncStorageSet)(unsafe.Pointer(C.Cgo_StorageSetFunc)),
						(C.FuncStorageDel)(unsafe.Pointer(C.Cgo_StorageDelFunc)))
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

func (sc *V8Engine) Execute(function, args string){

	cSource := C.CString(sc.source)
	defer C.free(unsafe.Pointer(cSource))
	C.InitializeSmartContract(cSource)

	functionCallScript := prepareFuncCallScript(function,args)
	cFunction := C.CString(functionCallScript)
	defer C.free(unsafe.Pointer(cFunction))

	handler := uint64(0)
	C.executeV8Script(cFunction, C.uintptr_t(handler))
}

func prepareFuncCallScript(function, args string) string{
	return fmt.Sprintf(`var instance = new _native_require();
						instance["%s"].apply(instance, [%s]);`,function, args)
}