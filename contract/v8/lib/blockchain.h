#include <v8.h>

using namespace v8;

void NewBlockchainInstance(Isolate *isolate, Local<Context> context, void *handler);
void VerifyAddressCallback(const FunctionCallbackInfo<Value> &info);
void TransferCallback(const FunctionCallbackInfo<Value> &info);
void GetCurrBlockHeightCallback(const FunctionCallbackInfo<Value> &info);
void GetNodeAddressCallback(const FunctionCallbackInfo<Value> &info);
void DeleteContractCallback(const FunctionCallbackInfo<Value> &info);