#ifndef _DAPPLEY_NF_VM_V8_LIB_EXECUTION_ENV_H_
#define _DAPPLEY_NF_VM_V8_LIB_EXECUTION_ENV_H_

#include <v8.h>

using namespace v8;

int SetupExecutionEnv(Isolate *isolate, Local<Context> &context);

#endif
