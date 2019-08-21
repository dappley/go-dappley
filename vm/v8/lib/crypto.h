#include <v8.h>

using namespace v8;

void NewCryptoInstance(Isolate *isolate, Local<Context> context, void *handler);
void VerifySignatureCallback(const FunctionCallbackInfo<Value> &info);
void VerifyPublicKeyCallback(const FunctionCallbackInfo<Value> &info);