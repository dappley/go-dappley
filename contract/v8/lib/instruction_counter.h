#ifndef _DAPPLEY_NF_VM_V8_LIB_INSTRUCTION_COUNTER_H_
#define _DAPPLEY_NF_VM_V8_LIB_INSTRUCTION_COUNTER_H_

#include <v8.h>

using namespace v8;

typedef void (*InstructionCounterIncrListener)(Isolate *isolate, size_t count,
                                               void *context);
void SetInstructionCounterIncrListener(InstructionCounterIncrListener listener);

void NewInstructionCounterInstance(Isolate *isolate, Local<Context> context,
                                   size_t *counter, void *listenerContext);

void IncrCounterCallback(const FunctionCallbackInfo<Value> &info);
void CountGetterCallback(Local<String> property,
                         const PropertyCallbackInfo<Value> &info);

void IncrCounter(Isolate *isolate, Local<Context> context, size_t count);

#endif
