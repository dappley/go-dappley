#define EXPORT __attribute__((__visibility__("default")))

#ifdef __cplusplus
extern "C" {
#endif
    EXPORT int executeV8Script(const char *sourceCode);
#ifdef __cplusplus
}
#endif