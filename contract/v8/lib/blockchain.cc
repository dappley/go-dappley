#include "blockchain.h"
#include "../engine.h"

static FuncVerifyAddress sVerifyAddress = NULL;
static FuncTransfer sTransfer = NULL;
static FuncGetCurrBlockHeight sGetCurrBlockHeight = NULL;


void InitializeBlockchain(FuncVerifyAddress verifyAddress, FuncTransfer transfer, FuncGetCurrBlockHeight getCurrBlockHeight){
  sVerifyAddress = verifyAddress;
  sTransfer = transfer;
  sGetCurrBlockHeight = getCurrBlockHeight;
}

void NewBlockchainInstance(Isolate *isolate, Local<Context> context, void *handler) {
  Local<ObjectTemplate> blockTpl = ObjectTemplate::New(isolate);
  blockTpl->SetInternalFieldCount(1);

  blockTpl->Set(String::NewFromUtf8(isolate, "verifyAddress"),
                FunctionTemplate::New(isolate, VerifyAddressCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  blockTpl->Set(String::NewFromUtf8(isolate, "transfer"),
                FunctionTemplate::New(isolate, TransferCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  blockTpl->Set(String::NewFromUtf8(isolate, "getCurrBlockHeight"),
                FunctionTemplate::New(isolate, GetCurrBlockHeightCallback),
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
  int ret = sVerifyAddress(*String::Utf8Value(isolate, address));
  info.GetReturnValue().Set(ret);

}

void TransferCallback(const FunctionCallbackInfo<Value> &info) {
  Isolate *isolate = info.GetIsolate();
  Local<Object> thisArg = info.Holder();
  Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

  if (info.Length() != 3) {
    isolate->ThrowException(String::NewFromUtf8(
        isolate, "Blockchain.transfer() requires 3 arguments"));
    return;
  }

  Local<Value> to = info[0];
  if (!to->IsString()) {
    isolate->ThrowException(
        String::NewFromUtf8(isolate, "to must be string"));
    return;
  }

  Local<Value> amount = info[1];
  if (!amount->IsString()) {
    isolate->ThrowException(
        String::NewFromUtf8(isolate, "amount must be string"));
    return;
  }

  Local<Value> tip = info[2];
  if (!tip->IsString()) {
    isolate->ThrowException(
        String::NewFromUtf8(isolate, "tip must be string"));
    return;
  }

  int ret = sTransfer(
    handler->Value(),
    *String::Utf8Value(isolate, to),
    *String::Utf8Value(isolate, amount),
    *String::Utf8Value(isolate, tip)
  );
  info.GetReturnValue().Set(ret);

}

void GetCurrBlockHeightCallback(const FunctionCallbackInfo<Value> &info) {
  Isolate *isolate = info.GetIsolate();
  Local<Object> thisArg = info.Holder();
  Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

  if (info.Length() != 0) {
    isolate->ThrowException(String::NewFromUtf8(
        isolate, "Blockchain.getCurrBlockHeight() does not require any argument"));
    return;
  }

  int ret = sGetCurrBlockHeight(handler->Value());
  info.GetReturnValue().Set(ret);

}
