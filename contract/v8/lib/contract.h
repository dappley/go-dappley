#include <v8.h>

using namespace v8;

void NewContractManagerInstance(Isolate *isolate, Local<Context> context, void *address);
void contractDeleteCallback(const FunctionCallbackInfo<Value> &info);
