package sc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ldappleyv8
#include <stdlib.h>
#include "v8/engine.h"
//blockchain
bool  Cgo_VerifyAddressFunc(const char *address);
//storage
char* Cgo_StorageGetFunc(void *address, const char *key);
int   Cgo_StorageSetFunc(void *address, const char *key, const char *value);
int   Cgo_StorageDelFunc(void *address, const char *key);
*/
import "C"
import (

	"fmt"
	"sync"
	"unsafe"

)

var (
	v8once 			= sync.Once{}
	v8EngineList 	= make(map[uint64]*V8Engine)
	storagesMutex 	= sync.RWMutex{}
	currHandler			= uint64(100)
)

type V8Engine struct{
	source  string
	storage map[string]string
	handler uint64
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
	engine := &V8Engine{
		source:  "",
		storage: make(map[string]string),
		handler: currHandler,
	}
	currHandler++;
	storagesMutex.Lock()
	defer storagesMutex.Unlock()
	v8EngineList[engine.handler] = engine
	return engine
}

func (sc *V8Engine) ImportSourceCode(source string){
	sc.source = source
}

func (sc *V8Engine) ImportLocalStorage(storage map[string]string){
	sc.storage = storage
}

func (sc *V8Engine) Execute(function, args string){

	cSource := C.CString(sc.source)
	defer C.free(unsafe.Pointer(cSource))
	C.InitializeSmartContract(cSource)

	functionCallScript := prepareFuncCallScript(function,args)
	cFunction := C.CString(functionCallScript)
	defer C.free(unsafe.Pointer(cFunction))

	C.executeV8Script(cFunction, C.uintptr_t(sc.handler))
}

func prepareFuncCallScript(function, args string) string{
	return fmt.Sprintf(`var instance = new _native_require();
						instance["%s"].apply(instance, [%s]);`,function, args)
}

func getV8EngineByAddress(handler uint64) *V8Engine{
	storagesMutex.Lock()
	defer storagesMutex.Unlock()
	return v8EngineList[handler]
}