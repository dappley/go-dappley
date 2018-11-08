#include <v8.h>

using namespace v8;

void NewStorageInstance(Isolate *isolate, Local<Context> context, void *address);
void storageGetCallback(const FunctionCallbackInfo<Value> &info);
void storageSetCallback(const FunctionCallbackInfo<Value> &info);
void storageDelCallback(const FunctionCallbackInfo<Value> &info);