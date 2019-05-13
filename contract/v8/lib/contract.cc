#include "contract.h"
#include "../engine.h"

static FuncContractDel cDel = NULL;

void InitializeContract(FuncContractDel del){
    cDel = del;
}

void NewContractInstance(Isolate *isolate, Local<Context> context, void *address){
    Local<ObjectTemplate> contractTpl = ObjectTemplate::New(isolate);
    contractTpl->SetInternalFieldCount(1);

    contractTpl->Set(String::NewFromUtf8(isolate, "del"),
                FunctionTemplate::New(isolate, contractDeleteCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));
    Local<Object> instance = contractTpl->NewInstance(context).ToLocalChecked();
    instance->SetInternalField(0, External::New(isolate, address));
    context->Global()->DefineOwnProperty(
      context, String::NewFromUtf8(isolate, "_contract"),
      instance,
      static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                     PropertyAttribute::ReadOnly));                                 
}

void contractDeleteCallback(const FunctionCallbackInfo<Value> &info){
    Isolate *isolate = info.GetIsolate();
    Local<Object> thisArg = info.Holder();
    Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

    if (info.Length() != 1) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "contract.del requires 1 arguments"));
        return;
    }

    Local<Value> key = info[0];
    if (!key->IsString()) {
        isolate->ThrowException(
            String::NewFromUtf8(isolate, "key must be string"));
        return;
    }
    int ret = cDel(handler->Value());

    info.GetReturnValue().Set(ret);

}