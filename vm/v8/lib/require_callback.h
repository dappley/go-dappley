#ifndef _DAPPLEY_NF_VM_V8_LIB_REQUIRE_CALLBACK_H_
#define _DAPPLEY_NF_VM_V8_LIB_REQUIRE_CALLBACK_H_

#include <v8.h>

using namespace v8;
#define LIB_WHITE "jslib/contract.js"

void NewNativeRequireFunction(Isolate *isolate, Local<ObjectTemplate> globalTpl);
void RequireCallback(const v8::FunctionCallbackInfo<v8::Value> &info);

#endif
