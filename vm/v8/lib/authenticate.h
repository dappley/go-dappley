#include <v8.h>

using namespace v8;

void NewAuthenCertInstance(Isolate *isolate, Local<Context> context, void *address);
void AuthenInitCallback(const FunctionCallbackInfo<Value> &info);
void AuthenVerifyCallback(const FunctionCallbackInfo<Value> &info);
