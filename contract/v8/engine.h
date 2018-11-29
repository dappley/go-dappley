#include <stdint.h>
#include <stdbool.h>
#include "lib/transaction_struct.h"
#include "lib/utxo_struct.h"

#define EXPORT __attribute__((__visibility__("default")))

#ifdef __cplusplus
extern "C" {
#endif
    typedef bool (*FuncVerifyAddress)(const char *address);
    typedef int (*FuncTransfer)(void *handler, const char *to, const char *amount, const char *tip);
    typedef char* (*FuncStorageGet)(void *address, const char *key);
    typedef int (*FuncStorageSet)(void *address, const char *key, const char *value);
    typedef int (*FuncStorageDel)(void *address, const char *key);
    typedef void (*FuncTransactionGet)(void* address, void* context);
    typedef void (*FuncPrevUtxoGet)(void* address, void* context);
    typedef void (*FuncLogger)(unsigned int level, char** args, int length);
    typedef int (*FuncRecordReward)(void *handler, const char *address, const char *amount);
    typedef bool (*FuncVerifySignature)(const char *msg, const char *pubKey, const char *sig);
    typedef int (*FuncRandom)(void *handler, int max);

    EXPORT void Initialize();
    EXPORT int executeV8Script(const char *sourceCode, uintptr_t handler, char **result);
    EXPORT void InitializeBlockchain(FuncVerifyAddress verifyAddress, FuncTransfer transfer);
    EXPORT void InitializeRewardDistributor(FuncRecordReward recordReward);
    EXPORT void InitializeStorage(FuncStorageGet get, FuncStorageSet set, FuncStorageDel del);
    EXPORT void InitializeTransaction(FuncTransactionGet get);
    EXPORT void InitializeCrypto(FuncVerifySignature verifySignature);
    EXPORT void InitializeMath(FuncRandom random);
    EXPORT void SetTransactionData(struct transaction_t* tx, void* context);
    EXPORT void InitializePrevUtxo(FuncPrevUtxoGet get);
    EXPORT void SetPrevUtxoData(struct utxo_t* utxos, int length, void* context);
    EXPORT void InitializeLogger(FuncLogger logger);
    EXPORT void InitializeSmartContract(char* source);
    EXPORT void DisposeV8();
#ifdef __cplusplus
}
#endif
