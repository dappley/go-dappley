#include <v8.h>

using namespace v8;

Local<ObjectTemplate> NewNativeRequireFunction(Isolate *isolate);
void LoadSmartContract(const v8::FunctionCallbackInfo<v8::Value> &info);
