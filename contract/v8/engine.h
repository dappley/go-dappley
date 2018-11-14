#include <stdint.h>
#include <stdbool.h>
#define EXPORT __attribute__((__visibility__("default")))

#ifdef __cplusplus
extern "C" {
#endif
    typedef bool (*FuncVerifyAddress)(const char *address);
    typedef int (*FuncTransfer)(void *handler, const char *to, const char *amount, const char *tip);
    typedef char* (*FuncStorageGet)(void *address, const char *key);
    typedef int (*FuncStorageSet)(void *address, const char *key, const char *value);
    typedef int (*FuncStorageDel)(void *address, const char *key);

    EXPORT void Initialize();
    EXPORT int executeV8Script(const char *sourceCode, uintptr_t handler, char **result);
    EXPORT void InitializeBlockchain(FuncVerifyAddress verifyAddress, FuncTransfer transfer);
    EXPORT void InitializeStorage(FuncStorageGet get, FuncStorageSet set, FuncStorageDel del);
    EXPORT void InitializeSmartContract(char* source);
    EXPORT void DisposeV8();
#ifdef __cplusplus
}
#endif
