#include "blockchain.h"
#include "../engine.h"

static VerifyAddressFunc sVerifyAddress = NULL;


void InitializeBlockchain(VerifyAddressFunc verifyAddress) {
  sVerifyAddress = verifyAddress;
}

void NewBlockchainInstance(Isolate *isolate, Local<Context> context, void *handler) {
  Local<ObjectTemplate> blockTpl = ObjectTemplate::New(isolate);
  blockTpl->SetInternalFieldCount(1);

  blockTpl->Set(String::NewFromUtf8(isolate, "verifyAddress"),
                FunctionTemplate::New(isolate, VerifyAddressCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  Local<Object> instance = blockTpl->NewInstance(context).ToLocalChecked();
  instance->SetInternalField(0, External::New(isolate, handler));

  context->Global()->DefineOwnProperty(
      context, String::NewFromUtf8(isolate, "_native_blockchain"), instance,
      static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                     PropertyAttribute::ReadOnly));
}

// VerifyAddressCallback
void VerifyAddressCallback(const FunctionCallbackInfo<Value> &info) {
    printf("VerifyAddressCallback\n");
    fflush(stdout);
  Isolate *isolate = info.GetIsolate();
  //Local<Object> thisArg = info.Holder();
  //Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

  if (info.Length() != 1) {
    isolate->ThrowException(String::NewFromUtf8(
        isolate, "Blockchain.verifyAddress() requires 1 arguments"));
    return;
  }

  Local<Value> address = info[0];
  if (!address->IsString()) {
    isolate->ThrowException(
        String::NewFromUtf8(isolate, "address must be string"));
    return;
  }

  int ret = 3;//sVerifyAddress(*String::Utf8Value(address->ToString()));
  info.GetReturnValue().Set(ret);

}