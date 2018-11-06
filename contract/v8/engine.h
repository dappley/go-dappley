#include <stdint.h>
#include <stdbool.h>
#define EXPORT __attribute__((__visibility__("default")))

#ifdef __cplusplus
extern "C" {
#endif
    typedef bool (*VerifyAddressFunc)(const char *address);

    EXPORT void Initialize();
    EXPORT int executeV8Script(const char *sourceCode, uintptr_t handler) ;
    EXPORT void InitializeBlockchain(VerifyAddressFunc verifyAddress);
    EXPORT void InitializeSmartContract(char* source);
    EXPORT void DisposeV8();
#ifdef __cplusplus
}
#endif

