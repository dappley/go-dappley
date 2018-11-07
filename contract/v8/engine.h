#include <stdint.h>
#include <stdbool.h>
#define EXPORT __attribute__((__visibility__("default")))

#ifdef __cplusplus
extern "C" {
#endif
    typedef bool (*FuncVerifyAddress)(const char *address);
    typedef char* (*FuncStorageGet)(const char *key);
    typedef int (*FuncStorageSet)(const char *key, const char *value);
    typedef int (*FuncStorageDel)(const char *key);

    EXPORT void Initialize();
    EXPORT int executeV8Script(const char *sourceCode, uintptr_t handler) ;
    EXPORT void InitializeBlockchain(FuncVerifyAddress verifyAddress);
    EXPORT void InitializeStorage(FuncStorageGet get, FuncStorageSet set, FuncStorageDel del);
    EXPORT void InitializeSmartContract(char* source);
    EXPORT void DisposeV8();
#ifdef __cplusplus
}
#endif

