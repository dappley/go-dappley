#include "load_sc.h"
#include "../engine.h"


char* scSource = NULL;

Local<ObjectTemplate> NewNativeRequireFunction(Isolate *isolate) {
  Local<ObjectTemplate> globalTpl = ObjectTemplate::New(isolate);
  globalTpl->SetInternalFieldCount(1);
  globalTpl->Set(String::NewFromUtf8(isolate, "_native_require"),
                 FunctionTemplate::New(isolate, LoadSmartContract),
                 static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                                PropertyAttribute::ReadOnly));
  return globalTpl;
}

void LoadSmartContract(const v8::FunctionCallbackInfo<v8::Value> &info){

    Isolate *isolate = info.GetIsolate();
    Local<Context> context = isolate->GetCurrentContext();
    Local<String> v8source = String::NewFromUtf8(isolate, scSource, NewStringType::kNormal).ToLocalChecked();
    MaybeLocal<Script> script = Script::Compile(context, v8source);
    MaybeLocal<Value> ret = script.ToLocalChecked()->Run(context);
    if (!ret.IsEmpty()) {
        Local<Value> rr = ret.ToLocalChecked();
        info.GetReturnValue().Set(rr);
    }
}

void InitializeSmartContract(char* source){
    scSource = source;
}