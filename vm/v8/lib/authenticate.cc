#include "authenticate.h"
#include "../engine.h"
#include "memory.h"

static FuncAuthenticateInitWithCert AuthenInit = NULL;
static FuncAuthenticateVerifyWithPublicKey AuthenVerify = NULL;

void InitializeAuthenCert(FuncAuthenticateInitWithCert authenInit, FuncAuthenticateVerifyWithPublicKey authenVerify) {
    AuthenInit = authenInit;
    AuthenVerify = authenVerify;
}

void NewAuthenCertInstance(Isolate *isolate, Local<Context> context, void *address) {
    Local<ObjectTemplate> storageTpl = ObjectTemplate::New(isolate);
    storageTpl->SetInternalFieldCount(1);

    storageTpl->Set(String::NewFromUtf8(isolate, "authenInit"), FunctionTemplate::New(isolate, AuthenInitCallback),
                    static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    storageTpl->Set(String::NewFromUtf8(isolate, "authenVerify"), FunctionTemplate::New(isolate, AuthenVerifyCallback),
                    static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));


    Local<Object> instance = storageTpl->NewInstance(context).ToLocalChecked();
    instance->SetInternalField(0, External::New(isolate, address));
    context->Global()->DefineOwnProperty(context, String::NewFromUtf8(isolate, "_native_authenticate_cert"), instance,
                                         static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));
}

// AuthenInitCallback
void AuthenInitCallback(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();
    Local<Object> thisArg = info.Holder();
    Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

    if (info.Length() != 1) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "Storage.Get requires 1 arguments"));
        return;
    }

    Local<Value> key = info[0];
    if (!key->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "key must be string"));
        return;
    }
    bool res = AuthenInit(handler->Value(), *String::Utf8Value(isolate, key));

    info.GetReturnValue().Set(res);
}

// AtuhenVerifyCallback
void AuthenVerifyCallback(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();
    Local<Object> thisArg = info.Holder();
    Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

    if (info.Length() != 0) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "Storage.Get requires 0 arguments"));
        return;
    }

    bool res = AuthenVerify(handler->Value());
    info.GetReturnValue().Set(res);
}
