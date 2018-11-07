#include "storage.h"
#include "../engine.h"

static FuncStorageGet sGet = NULL;
static FuncStorageSet sSet = NULL;
static FuncStorageDel sDel = NULL;

void InitializeStorage(FuncStorageGet get, FuncStorageSet set, FuncStorageDel del) {
  sGet = get;
  sSet = set;
  sDel = del;
}

void NewStorageInstance(Isolate *isolate, Local<Context> context) {
  Local<ObjectTemplate> storageTpl = ObjectTemplate::New(isolate);
  storageTpl->SetInternalFieldCount(1);

  storageTpl->Set(String::NewFromUtf8(isolate, "get"),
                FunctionTemplate::New(isolate, storageGetCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  storageTpl->Set(String::NewFromUtf8(isolate, "set"),
                FunctionTemplate::New(isolate, storageSetCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  storageTpl->Set(String::NewFromUtf8(isolate, "del"),
                FunctionTemplate::New(isolate, storageDelCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  Local<Object> handlers = storageTpl->NewInstance(context).ToLocalChecked();

  context->Global()->DefineOwnProperty(
      context, String::NewFromUtf8(isolate, "_native_storage"),
      handlers,
      static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                     PropertyAttribute::ReadOnly));
}

// storageGetCallback
void storageGetCallback(const FunctionCallbackInfo<Value> &info){
    Isolate *isolate = info.GetIsolate();

    if (info.Length() != 1) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "Storage.Get requires 1 arguments"));
        return;
    }

    Local<Value> key = info[0];
    if (!key->IsString()) {
        isolate->ThrowException(
            String::NewFromUtf8(isolate, "key must be string"));
        return;
    }
    char *res = sGet(*String::Utf8Value(isolate, key));

    if (res == NULL) {
        info.GetReturnValue().SetNull();
    } else {
        info.GetReturnValue().Set(String::NewFromUtf8(isolate, res));
        free(res);
    }
}

// storageSetCallback
void storageSetCallback(const FunctionCallbackInfo<Value> &info){
    Isolate *isolate = info.GetIsolate();

    if (info.Length() != 2) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "Storage.Set requires 2 arguments"));
        return;
    }

    Local<Value> key = info[0];
    if (!key->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "key must be string"));
        return;
    }

    Local<Value> value = info[1];
    if (!value->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "value must be string"));
        return;
    }

    int ret = sSet(*String::Utf8Value(isolate, key),*String::Utf8Value(isolate, value));

    info.GetReturnValue().Set(ret);
}

// storageDelCallback
void storageDelCallback(const FunctionCallbackInfo<Value> &info){
    Isolate *isolate = info.GetIsolate();

    if (info.Length() != 1) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "Storage.Del requires 1 arguments"));
        return;
    }

    Local<Value> key = info[0];
    if (!key->IsString()) {
        isolate->ThrowException(
            String::NewFromUtf8(isolate, "key must be string"));
        return;
    }
    int ret = sDel(*String::Utf8Value(isolate, key));

    info.GetReturnValue().Set(ret);
}