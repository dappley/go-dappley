#include "crypto.h"
#include "../engine.h"

static FuncVerifySignature sVerifySignature = NULL;
static FuncVerifyPublicKey sVerifyPublicKey = NULL;

void InitializeCrypto(FuncVerifySignature verifySignature, FuncVerifyPublicKey verifyPublicKey) {
    sVerifySignature = verifySignature;
    sVerifyPublicKey = verifyPublicKey;
}

void NewCryptoInstance(Isolate *isolate, Local<Context> context, void *handler) {
    Local<ObjectTemplate> cryptoTpl = ObjectTemplate::New(isolate);
    cryptoTpl->SetInternalFieldCount(1);

    cryptoTpl->Set(String::NewFromUtf8(isolate, "verifySignature"), FunctionTemplate::New(isolate, VerifySignatureCallback),
                   static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    cryptoTpl->Set(String::NewFromUtf8(isolate, "verifyPublicKey"), FunctionTemplate::New(isolate, VerifyPublicKeyCallback),
                   static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    Local<Object> instance = cryptoTpl->NewInstance(context).ToLocalChecked();
    instance->SetInternalField(0, External::New(isolate, handler));

    context->Global()->DefineOwnProperty(context, String::NewFromUtf8(isolate, "crypto"), instance,
                                         static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));
}

void VerifySignatureCallback(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();

    if (info.Length() != 3) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "crypto.verifySignature() requires 3 arguments"));
        return;
    }

    Local<Value> msg = info[0];
    if (!msg->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "message must be string"));
        return;
    }

    Local<Value> pubKey = info[1];
    if (!pubKey->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "public key must be string"));
        return;
    }

    Local<Value> sig = info[2];
    if (!sig->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "signature must be string"));
        return;
    }

    bool ret = sVerifySignature(*String::Utf8Value(isolate, msg), *String::Utf8Value(isolate, pubKey), *String::Utf8Value(isolate, sig));

    info.GetReturnValue().Set(ret);
}

void VerifyPublicKeyCallback(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();

    if (info.Length() != 2) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "crypto.verifyPublicKey() requires 2 arguments"));
        return;
    }

    Local<Value> addr = info[0];
    if (!addr->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "address must be string"));
        return;
    }

    Local<Value> pubKey = info[1];
    if (!pubKey->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "public key must be string"));
        return;
    }

    bool ret = sVerifyPublicKey(*String::Utf8Value(isolate, addr), *String::Utf8Value(isolate, pubKey));

    info.GetReturnValue().Set(ret);
}