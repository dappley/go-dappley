package vm

/*
#cgo LDFLAGS: -L${SRCDIR}/lib -ldappleyv8
#include <stdlib.h>
#include "v8/engine.h"

char *Cgo_RequireDelegateFunc(void *handler, const char *filename, size_t *lineOffset);
char *Cgo_AttachLibVersionDelegateFunc(void *handler, const char *libname);

//blockchain
bool  Cgo_VerifyAddressFunc(const char *address, size_t *gasCnt);
int	  Cgo_TransferFunc(void *handler, const char *to, const char *amount, const char *tip, size_t *gasCnt);
int   Cgo_GetCurrBlockHeightFunc(void *handler);
char* Cgo_GetNodeAddressFunc(void *handler);
int   Cgo_DeleteContractFunc(void *handler);
//storage
char* Cgo_StorageGetFunc(void *address, const char *key);
int   Cgo_StorageSetFunc(void *address, const char *key, const char *value);
int   Cgo_StorageDelFunc(void *address, const char *key);
//authen cert
bool Cgo_AuthenticateInitWithCertFunc(void *address , const char *cert);
bool Cgo_AuthenticateVerifyWithPublicKeyFunc(void *address);

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
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/utxo"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/byteutils"
	"github.com/dappley/go-dappley/crypto/hash"
	lru "github.com/hashicorp/golang-lru"

	logger "github.com/sirupsen/logrus"
)

var (
	v8once               = sync.Once{}
	v8EngineList         = make(map[uint64]*V8Engine)
	currHandler          = uint64(100)
	sourceModuleCache, _ = lru.New(40960)
	enginesLock          = sync.RWMutex{}
)

const (
	TimeoutGasLimitCost              = 100000000
	MaxLimitsOfExecutionInstructions = 10000000 // TODO: set max gasLimit with execution 5s *0.8
)

type V8Engine struct {
	source             string
	state              *scState.ScState
	tx                 *transaction.Transaction
	rewards            map[string]string
	contractAddr       account.Address
	contractCreateUTXO *utxo.UTXO
	contractUTXOs      []*utxo.UTXO
	prevUtxos          []*utxo.UTXO
	sourceTXID         []byte
	generatedTXs       []*transaction.Transaction
	handler            uint64
	blkHeight          uint64
	seed               int64
	nodeAddr           account.Address

	modules                                 Modules
	v8engine                                *C.V8Engine
	strictDisallowUsageOfInstructionCounter int
	enableLimits                            bool
	limitsOfExecutionInstructions           uint64
	limitsOfTotalMemorySize                 uint64
	actualCountOfExecutionInstructions      uint64
	actualTotalMemorySize                   uint64
	innerErrMsg                             string
	innerErr                                error
}

type sourceModuleItem struct {
	source                    string
	sourceLineOffset          int
	traceableSource           string
	traceableSourceLineOffset int
}

func InitializeV8Engine() {
	C.Initialize()
	// Require.
	C.InitializeRequireDelegate((C.RequireDelegate)(unsafe.Pointer(C.Cgo_RequireDelegateFunc)), (C.AttachLibVersionDelegate)(unsafe.Pointer(C.Cgo_AttachLibVersionDelegateFunc)))
	// execution_env require
	C.InitializeExecutionEnvDelegate((C.AttachLibVersionDelegate)(unsafe.Pointer(C.Cgo_AttachLibVersionDelegateFunc)))
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
	C.InitializeAuthenCert(
		(C.FuncAuthenticateInitWithCert)(unsafe.Pointer(C.Cgo_AuthenticateInitWithCertFunc)),
		(C.FuncAuthenticateVerifyWithPublicKey)(unsafe.Pointer(C.Cgo_AuthenticateVerifyWithPublicKeyFunc)))
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
		source:                             "",
		state:                              nil,
		tx:                                 nil,
		contractAddr:                       account.NewAddress(""),
		handler:                            currHandler,
		modules:                            NewModules(),
		v8engine:                           C.CreateEngine(),
		enableLimits:                       true,
		limitsOfExecutionInstructions:      0,
		limitsOfTotalMemorySize:            0,
		actualCountOfExecutionInstructions: 0,
		actualTotalMemorySize:              0,
	}
	currHandler++
	enginesLock.Lock()
	defer enginesLock.Unlock()
	v8EngineList[engine.handler] = engine
	return engine
}

//DestroyEngine destroy V8Engine instance
func (sc *V8Engine) DestroyEngine() {
	enginesLock.Lock()
	delete(v8EngineList, sc.handler)
	enginesLock.Unlock()

	C.DeleteEngine(sc.v8engine)
}

func (sc *V8Engine) ImportSourceCode(source string) {
	sc.source = source
}

func (sc *V8Engine) ImportLocalStorage(state *scState.ScState) {
	sc.state = state
}

func (sc *V8Engine) ImportTransaction(tx *transaction.Transaction) {
	sc.tx = tx
}

func (sc *V8Engine) ImportContractCreateUTXO(utxo *utxo.UTXO) {
	sc.contractCreateUTXO = utxo
}

// ImportContractAddr supplies the account address of the contract to the engine
func (sc *V8Engine) ImportContractAddr(contractAddr account.Address) {
	sc.contractAddr = contractAddr
}

// ImportUTXOs supplies the list of contract's UTXOs to the engine
func (sc *V8Engine) ImportUTXOs(utxos []*utxo.UTXO) {
	sc.contractUTXOs = make([]*utxo.UTXO, len(utxos))
	copy(sc.contractUTXOs, utxos)
}

// ImportSourceTXID supplies the id of the transaction which executes the contract
func (sc *V8Engine) ImportSourceTXID(txid []byte) {
	sc.sourceTXID = txid
}

// GetGeneratedTXs returns the transactions generated as a result of executing the contract
func (sc *V8Engine) GetGeneratedTXs() []*transaction.Transaction {
	return sc.generatedTXs
}

func (sc *V8Engine) ImportRewardStorage(rewards map[string]string) {
	sc.rewards = rewards
}

// ImportPrevUtxos supplies the utxos of vin in current transaction
func (sc *V8Engine) ImportPrevUtxos(utxos []*utxo.UTXO) {
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
func (sc *V8Engine) ImportNodeAddress(addr account.Address) {
	sc.nodeAddr = addr
}

// ClearModuleCache ..
func ClearSourceModuleCache() {
	sourceModuleCache.Purge()
}

// AddModule add module.
func (sc *V8Engine) AddModule(id, source string, sourceLineOffset int) error {
	// inject tracing instruction when enable limits.
	if sc.enableLimits {
		var item *sourceModuleItem
		sourceHash := byteutils.Hex(hash.Sha3256([]byte(source)))

		// try read from cache.
		if sourceModuleCache.Contains(sourceHash) {
			value, _ := sourceModuleCache.Get(sourceHash)
			item = value.(*sourceModuleItem)
		}
		if item == nil {
			traceableSource, lineOffset, err := sc.InjectTracingInstructions(source)
			if err != nil {
				logger.WithFields(logger.Fields{
					"err": err,
				}).Debug("Failed to inject tracing instruction.")
				return err
			}

			item = &sourceModuleItem{
				source:                    source,
				sourceLineOffset:          sourceLineOffset,
				traceableSource:           traceableSource,
				traceableSourceLineOffset: lineOffset,
			}
			// put to cache.
			sourceModuleCache.Add(sourceHash, item)
		}

		source = item.traceableSource
		sourceLineOffset = item.traceableSourceLineOffset
	}
	sc.modules.Add(NewModule(id, source, sourceLineOffset))
	return nil
}

func (sc *V8Engine) Execute(function, args string) (string, error) {
	var err error

	cSource := C.CString(sc.source)
	defer C.free(unsafe.Pointer(cSource))
	C.InitializeSmartContract(cSource)

	var runnableSource string
	var sourceLineOffset int
	runnableSource, sourceLineOffset, err = sc.prepareFuncCallScript(sc.source, function, args)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	sc.CollectTracingStats()
	mem := sc.actualTotalMemorySize + DefaultLimitsOfTotalMemorySize
	if err := sc.SetExecutionLimits(sc.limitsOfExecutionInstructions, mem); err != nil {
		logger.Error(err)
		return "", err
	}
	// set max
	if sc.limitsOfExecutionInstructions > MaxLimitsOfExecutionInstructions {
		sc.SetExecutionLimits(MaxLimitsOfExecutionInstructions, sc.limitsOfTotalMemorySize)
	}

	result, err := sc.RunScriptSource(runnableSource, sourceLineOffset)

	if sc.limitsOfExecutionInstructions == MaxLimitsOfExecutionInstructions && err == ErrInsufficientGas {
		err = ErrExecutionTimeout
		result = "null"
	}
	return result, err
}

// RunScriptSource run js source.
func (sc *V8Engine) RunScriptSource(runnableSource string, sourceLineOffset int) (string, error) {
	var (
		result  string
		err     error
		ret     C.int
		cResult *C.char
	)
	cFunction := C.CString(runnableSource)
	defer C.free(unsafe.Pointer(cFunction))
	ret = C.executeV8Script(cFunction, C.int(sourceLineOffset), C.uintptr_t(sc.handler), &cResult, sc.v8engine)
	sc.CollectTracingStats()

	if sc.innerErr != nil {
		if sc.innerErrMsg == "" { //the first call of muti-vm
			result = "Inner Contract: \"\""
		} else {
			result = "Inner Contract: " + sc.innerErrMsg
		}
		err := sc.innerErr
		if cResult != nil {
			C.free(unsafe.Pointer(cResult))
		}
		if sc.actualCountOfExecutionInstructions > sc.limitsOfExecutionInstructions {
			sc.actualCountOfExecutionInstructions = sc.limitsOfExecutionInstructions
		}
		return result, err
	}

	if ret == C.VM_EXE_TIMEOUT_ERR {
		err = ErrExecutionTimeout
		if TimeoutGasLimitCost > sc.limitsOfExecutionInstructions {
			sc.actualCountOfExecutionInstructions = sc.limitsOfExecutionInstructions
		} else {
			sc.actualCountOfExecutionInstructions = TimeoutGasLimitCost
		}
	} else if ret == C.VM_UNEXPECTED_ERR {
		err = ErrUnexpected
	} else if ret == C.VM_INNER_EXE_ERR {
		err = ErrInnerExecutionFailed
		if sc.limitsOfExecutionInstructions < sc.actualCountOfExecutionInstructions {
			logger.WithFields(logger.Fields{
				"actualGas": sc.actualCountOfExecutionInstructions,
				"limitGas":  sc.limitsOfExecutionInstructions,
			}).Error("Unexpected error: actual gas exceed the limit")
		}
	} else {
		if ret != C.VM_SUCCESS {
			err = ErrExecutionFailed
		}
		if sc.limitsOfExecutionInstructions > 0 &&
			sc.limitsOfExecutionInstructions < sc.actualCountOfExecutionInstructions {
			// Reach instruction limits.
			err = ErrInsufficientGas
			sc.actualCountOfExecutionInstructions = sc.limitsOfExecutionInstructions
		} else if sc.limitsOfTotalMemorySize > 0 && sc.limitsOfTotalMemorySize < sc.actualTotalMemorySize {
			// reach memory limits.
			err = ErrExceedMemoryLimits
			sc.actualCountOfExecutionInstructions = sc.limitsOfExecutionInstructions
		}
	}

	if cResult != nil {
		result = C.GoString(cResult)
		C.free(unsafe.Pointer(cResult))
	} else if ret == C.VM_SUCCESS {
		result = ""
	}

	return result, err
}

func (sc *V8Engine) CheckContactSyntax(source string) error {

	cSource := C.CString(source)
	defer C.free(unsafe.Pointer(cSource))
	var err error = nil
	if C.CheckContractSyntax(cSource, sc.v8engine) > 0 {
		err = errors.New("error syntax")
	}
	return err
}

func (sc *V8Engine) prepareFuncCallScript(source, function, args string) (string, int, error) {
	sourceLineOffset := 0

	// add module.
	const ModuleID string = "contract.js"
	if err := sc.AddModule(ModuleID, source, sourceLineOffset); err != nil {
		logger.WithFields(logger.Fields{
			"ModuleID": ModuleID,
		}).Error(err)
		return "", 0, err
	}
	var runnableSource string
	runnableSource = fmt.Sprintf(`var __instance = require("%s");__instance["%s"].apply(__instance, [%s]);`, ModuleID, function, args)
	return runnableSource, 0, nil
}

func getV8EngineByAddress(handler uint64) *V8Engine {
	enginesLock.Lock()
	defer enginesLock.Unlock()
	return v8EngineList[handler]
}

// CollectTracingStats collect tracing data from v8 engine.
func (sc *V8Engine) CollectTracingStats() {
	// read memory stats.
	C.ReadMemoryStatistics(sc.v8engine)

	sc.actualCountOfExecutionInstructions = uint64(sc.v8engine.stats.count_of_executed_instructions)
	sc.actualTotalMemorySize = uint64(sc.v8engine.stats.total_memory_size)
}

// SetExecutionLimits set execution limits of V8 Engine, prevent Halting Problem.
func (sc *V8Engine) SetExecutionLimits(limitsOfExecutionInstructions, limitsOfTotalMemorySize uint64) error {

	totalMemorySize := DefaultLimitsOfTotalMemorySize
	if limitsOfTotalMemorySize > 0 {
		totalMemorySize = limitsOfTotalMemorySize
	}

	sc.v8engine.limits_of_executed_instructions = C.size_t(limitsOfExecutionInstructions)
	sc.v8engine.limits_of_total_memory_size = C.size_t(totalMemorySize)

	sc.limitsOfExecutionInstructions = limitsOfExecutionInstructions
	sc.limitsOfTotalMemorySize = totalMemorySize

	if limitsOfExecutionInstructions == 0 || totalMemorySize == 0 {
		logger.Errorf("limit args has empty. limitsOfExecutionInstructions:%v,limitsOfTotalMemorySize:%d", limitsOfExecutionInstructions, totalMemorySize)
		return ErrLimitHasEmpty
	}
	// V8 needs at least 6M heap memory.
	if totalMemorySize > 0 && totalMemorySize < 6000000 {
		logger.Errorf("V8 needs at least 6M (6000000) heap memory, your limitsOfTotalMemorySize (%d) is too low.", totalMemorySize)
		return ErrSetMemorySmall
	}
	return nil
}

// InjectTracingInstructions process the source to inject tracing instructions.
func (sc *V8Engine) InjectTracingInstructions(source string) (string, int, error) {
	cSource := C.CString(source)
	defer C.free(unsafe.Pointer(cSource))

	lineOffset := C.int(0)
	traceableCSource := C.RunInjectTracingInstructionsThread(sc.v8engine, cSource, &lineOffset, C.int(sc.strictDisallowUsageOfInstructionCounter), C.uintptr_t(sc.handler))
	if traceableCSource == nil {
		return "", 0, ErrInjectTracingInstructionFailed
	}

	defer C.free(unsafe.Pointer(traceableCSource))
	return C.GoString(traceableCSource), int(lineOffset), nil
}

// ExecutionInstructions returns the execution instructions
func (sc *V8Engine) ExecutionInstructions() uint64 {
	return sc.actualCountOfExecutionInstructions
}
