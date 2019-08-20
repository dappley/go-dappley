#include <v8.h>

using namespace v8;

void NewRewardDistributorInstance(Isolate *isolate, Local<Context> context, void *handler);
void RecordRewardCallback(const FunctionCallbackInfo<Value> &info);
