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
#ifdef __cplusplus
}
#endif

