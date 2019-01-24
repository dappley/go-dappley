#include <v8.h>

using namespace v8;

void NewEventInstance(Isolate *isolate, Local<Context> context, void *address);
void triggerEventCallback(const FunctionCallbackInfo<Value> &info);