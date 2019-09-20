#include "memory.h"
#include "../engine.h"

static FuncMalloc sMalloc = NULL;
static FuncFree sFree = NULL;

void InitializeMemoryFunc(FuncMalloc mallocFunc, FuncFree freeFunc) {
    sMalloc = mallocFunc;
    sFree = freeFunc;
}

void* MyMalloc(size_t size) {
    if (sMalloc != NULL) {
        return sMalloc(size);
    }
    return malloc(size);
}

void MyFree(void* data) {
    if (sFree != NULL) {
        return sFree(data);
    }
    return free(data);
}