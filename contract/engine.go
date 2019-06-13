package vm

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ldappleyv8
#include <stdlib.h>
#include "v8/engine.h"
//blockchain
bool  Cgo_VerifyAddressFunc(const char *address);
int	  Cgo_TransferFunc(void *handler, const char *to, const char *amount, const char *tip);
int   Cgo_GetCurrBlockHeightFunc(void *handler);
char* Cgo_GetNodeAddressFunc(void *handler);
int   Cgo_DeleteContractFunc(void *handler);
//storage
char* Cgo_StorageGetFunc(void *address, const char *key);
int   Cgo_StorageSetFunc(void *address, const char *key, const char *value);
int   Cgo_StorageDelFunc(void *address, const char *key);
int   Cgo_TriggerEventFunc(void *address, const char *topic, const char *data);
int	  Cgo_RecordRewardFunc(void *handler, const char *address, const char *amount);
//transaction
struct transaction_t* Cgo_TransactionGetFunc(void *address);
//log
void Cgo_LoggerFunc(unsigned int level, const char ** args, int length);
//prev utxo
void Cgo_PrevUtxoGetFunc(void *address, void* context);
//crypto
bool Cgo_VerifySignatureFunc(const char *msg, const char *pubkey, const char *sig);
bool Cgo_VerifyPublicKeyFunc(const char *addr, const char *pubkey);
//math
int Cgo_RandomFunc(void *handler, int max);

void* Cgo_Malloc(size_t size);
void  Cgo_Free(void* address);
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

var (
	v8once        = sync.Once{}
	v8EngineList  = make(map[uint64]*V8Engine)
	storagesMutex = sync.RWMutex{}
	currHandler   = uint64(100)
)

type V8Engine struct {
	source        string
	state         *core.ScState
	tx            *core.Transaction
	rewards       map[string]string
	contractAddr  core.Address
	contractUTXOs []*core.UTXO
	prevUtxos     []*core.UTXO
	sourceTXID    []byte
	generatedTXs  []*core.Transaction
	handler       uint64
	blkHeight     uint64
	seed          int64
	nodeAddr      core.Address
}

func InitializeV8Engine() {
	C.Initialize()
	C.InitializeBlockchain(
		(C.FuncVerifyAddress)(unsafe.Pointer(C.Cgo_VerifyAddressFunc)),
		(C.FuncTransfer)(unsafe.Pointer(C.Cgo_TransferFunc)),
		(C.FuncGetCurrBlockHeight)(unsafe.Pointer(C.Cgo_GetCurrBlockHeightFunc)),
		(C.FuncGetNodeAddress)(unsafe.Pointer(C.Cgo_GetNodeAddressFunc)),
		(C.FuncDeleteContract)(unsafe.Pointer(C.Cgo_DeleteContractFunc)),
	)
	C.InitializeStorage(
		(C.FuncStorageGet)(unsafe.Pointer(C.Cgo_StorageGetFunc)),
		(C.FuncStorageSet)(unsafe.Pointer(C.Cgo_StorageSetFunc)),
		(C.FuncStorageDel)(unsafe.Pointer(C.Cgo_StorageDelFunc)))
	C.InitializeTransaction((C.FuncTransactionGet)(unsafe.Pointer(C.Cgo_TransactionGetFunc)))
	C.InitializeLogger((C.FuncLogger)(unsafe.Pointer(C.Cgo_LoggerFunc)))
	C.InitializeRewardDistributor((C.FuncRecordReward)(unsafe.Pointer(C.Cgo_RecordRewardFunc)))
	C.InitializeEvent((C.FuncTriggerEvent)(unsafe.Pointer(C.Cgo_TriggerEventFunc)))
	C.InitializePrevUtxo((C.FuncPrevUtxoGet)(unsafe.Pointer(C.Cgo_PrevUtxoGetFunc)))
	C.InitializeCrypto(
		(C.FuncVerifySignature)(unsafe.Pointer(C.Cgo_VerifySignatureFunc)),
		(C.FuncVerifyPublicKey)(unsafe.Pointer(C.Cgo_VerifyPublicKeyFunc)),
	)
	C.InitializeMath((C.FuncRandom)(unsafe.Pointer(C.Cgo_RandomFunc)))
	C.InitializeMemoryFunc((C.FuncMalloc)(unsafe.Pointer(C.Cgo_Malloc)), (C.FuncFree)(unsafe.Pointer(C.Cgo_Free)))
}

//NewV8Engine generates a new V8Engine instance
func NewV8Engine() *V8Engine {
	v8once.Do(func() { InitializeV8Engine() })
	engine := &V8Engine{
		source:       "",
		state:        nil,
		tx:           nil,
		contractAddr: core.NewAddress(""),
		handler:      currHandler,
	}
	currHandler++
	storagesMutex.Lock()
	defer storagesMutex.Unlock()
	v8EngineList[engine.handler] = engine
	return engine
}

//DestroyEngine destroy V8Engine instance
func (sc *V8Engine) DestroyEngine() {
	storagesMutex.Lock()
	defer storagesMutex.Unlock()
	delete(v8EngineList, sc.handler)
}

func (sc *V8Engine) ImportSourceCode(source string) {
	sc.source = source
}

func (sc *V8Engine) ImportLocalStorage(state *core.ScState) {
	sc.state = state
}

func (sc *V8Engine) ImportTransaction(tx *core.Transaction) {
	sc.tx = tx
}

// ImportContractAddr supplies the wallet address of the contract to the engine
func (sc *V8Engine) ImportContractAddr(contractAddr core.Address) {
	sc.contractAddr = contractAddr
}

// ImportUTXOs supplies the list of contract's UTXOs to the engine
func (sc *V8Engine) ImportUTXOs(utxos []*core.UTXO) {
	sc.contractUTXOs = make([]*core.UTXO, len(utxos))
	copy(sc.contractUTXOs, utxos)
}

// ImportSourceTXID supplies the id of the transaction which executes the contract
func (sc *V8Engine) ImportSourceTXID(txid []byte) {
	sc.sourceTXID = txid
}

// GetGeneratedTXs returns the transactions generated as a result of executing the contract
func (sc *V8Engine) GetGeneratedTXs() []*core.Transaction {
	return sc.generatedTXs
}

func (sc *V8Engine) ImportRewardStorage(rewards map[string]string) {
	sc.rewards = rewards
}

// ImportPrevUtxos supplies the utxos of vin in current transaction
func (sc *V8Engine) ImportPrevUtxos(utxos []*core.UTXO) {
	sc.prevUtxos = utxos
}

// ImportCurrBlockHeight imports the current block height
func (sc *V8Engine) ImportCurrBlockHeight(blkHeight uint64) {
	sc.blkHeight = blkHeight
}

// ImportCurrBlockHeight imports the current block height
func (sc *V8Engine) ImportSeed(seed int64) {
	sc.seed = seed
}

// ImportCurrBlockHeight imports the current block height
func (sc *V8Engine) ImportNodeAddress(addr core.Address) {
	sc.nodeAddr = addr
}

func (sc *V8Engine) Execute(function, args string) string {
	res := "\"\""
	status := "success"
	var result *C.char

	cSource := C.CString(sc.source)
	defer C.free(unsafe.Pointer(cSource))
	C.InitializeSmartContract(cSource)

	functionCallScript := prepareFuncCallScript(function, args)
	cFunction := C.CString(functionCallScript)
	defer C.free(unsafe.Pointer(cFunction))

	if C.executeV8Script(cFunction, C.uintptr_t(sc.handler), &result) > 0 {
		status = "failed"
	}

	if result != nil {
		res = C.GoString(result)
		C.free(unsafe.Pointer(result))
	}

	logger.WithFields(logger.Fields{
		"result": res,
		"status": status,
	}).Info("V8Engine: smart contract execution ends.")
	return res
}

func prepareFuncCallScript(function, args string) string {
	return fmt.Sprintf(
		`var instance = new _native_require();instance["%s"].apply(instance, [%s]);`,
		function,
		args,
	)
}

func getV8EngineByAddress(handler uint64) *V8Engine {
	storagesMutex.Lock()
	defer storagesMutex.Unlock()
	return v8EngineList[handler]
}
