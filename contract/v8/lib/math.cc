#include "math.h"
#include "../engine.h"

static FuncRandom sRandom = NULL;

void InitializeMath(FuncRandom random) {
  sRandom = random;
}

void NewMathInstance(Isolate *isolate, Local<Context> context, void *handler) {
  Local<ObjectTemplate> mathTpl = ObjectTemplate::New(isolate);
  mathTpl->SetInternalFieldCount(1);

  mathTpl->Set(String::NewFromUtf8(isolate, "random"),
                FunctionTemplate::New(isolate, RandomCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

  Local<Object> instance = mathTpl->NewInstance(context).ToLocalChecked();
  instance->SetInternalField(0, External::New(isolate, handler));

  context->Global()->DefineOwnProperty(
      context, String::NewFromUtf8(isolate, "math"), instance,
      static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                     PropertyAttribute::ReadOnly));
}

void RandomCallback(const FunctionCallbackInfo<Value> &info) {

  Isolate *isolate = info.GetIsolate();
  Local<Object> thisArg = info.Holder();
  Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

  if (info.Length() != 1) {
    isolate->ThrowException(String::NewFromUtf8(
        isolate, "math.random() requires 1 argument"));
    return;
  }

  Local<Value> max = info[0];
  if (!max->IsNumber()) {
    isolate->ThrowException(
        String::NewFromUtf8(isolate, "input must be a number"));
    return;
  }

  int ret = sRandom(handler->Value(), Number::Cast(*max)->Value());

  info.GetReturnValue().Set(ret);
}
