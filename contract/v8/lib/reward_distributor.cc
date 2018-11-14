#include "reward_distributor.h"
#include "../engine.h"

static FuncRecordReward sRecordReward = NULL;


void InitializeRewardDistributor(FuncRecordReward recordReward) {
  sRecordReward = recordReward;
}

void NewRewardDistributorInstance(Isolate *isolate, Local<Context> context, void *handler) {
  Local<ObjectTemplate> rdTpl = ObjectTemplate::New(isolate);
  rdTpl->SetInternalFieldCount(1);

  rdTpl->Set(String::NewFromUtf8(isolate, "record"),
                FunctionTemplate::New(isolate, RecordRewardCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  Local<Object> instance = rdTpl->NewInstance(context).ToLocalChecked();
  instance->SetInternalField(0, External::New(isolate, handler));

  context->Global()->DefineOwnProperty(
      context, String::NewFromUtf8(isolate, "_native_reward"), instance,
      static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                     PropertyAttribute::ReadOnly));
}

// VerifyAddressCallback
void RecordRewardCallback(const FunctionCallbackInfo<Value> &info) {

    printf("RecordRewardCallbaack!");
    fflush(stdout);

    Isolate *isolate = info.GetIsolate();
    Local<Object> thisArg = info.Holder();
    Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

    if (info.Length() != 2) {
        isolate->ThrowException(String::NewFromUtf8(
            isolate, "Blockchain.verifyAddress() requires 2 arguments"));
        return;
    }

    Local<Value> address = info[0];
    if (!address->IsString()) {
        isolate->ThrowException(
            String::NewFromUtf8(isolate, "address must be string"));
        return;
    }

    Local<Value> amount = info[1];
    if (!address->IsString()) {
        isolate->ThrowException(
          String::NewFromUtf8(isolate, "address must be string"));
        return;
    }

    int ret = sRecordReward(handler->Value(), *String::Utf8Value(isolate, address),*String::Utf8Value(isolate, amount));
    info.GetReturnValue().Set(ret);
}