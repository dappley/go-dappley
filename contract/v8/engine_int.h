#ifndef _DAPPLEY_NF_VM_V8_ENGINE_INT_H_
#define _DAPPLEY_NF_VM_V8_ENGINE_INT_H_

#include "engine.h"

#include <v8.h>

using namespace v8;

typedef int (*ExecutionDelegate)(char **result, Isolate *isolate, const char *source, int source_line_offset, Local<Context> context, TryCatch &trycatch,
                                 void *delegateContext);

int ExecuteDelegate(const char *sourceCode, int source_line_offset, uintptr_t handler, char **result, V8Engine *e, ExecutionDelegate delegate, void *delegateContext);

int ExecuteSourceDataDelegate(char **result, Isolate *isolate, const char *source, int source_line_offset, Local<Context> context, TryCatch &trycatch,
                              void *delegateContext);

#endif
