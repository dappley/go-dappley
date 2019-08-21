#include <v8.h>

using namespace v8;

void NewMathInstance(Isolate *isolate, Local<Context> context, void *handler);
void RandomCallback(const FunctionCallbackInfo<Value> &info);