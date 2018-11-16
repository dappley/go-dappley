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
  Local<Script> script;
  if (!Script::Compile(context, v8source).ToLocal(&script)) {
    // compilation error
    return;
  }
  Local<Value> ret;
  if (!script->Run(context).ToLocal(&ret)) {
    // runtime error
    return;
  }
  info.GetReturnValue().Set(ret);
}

void InitializeSmartContract(char* source){
  scSource = source;
}
