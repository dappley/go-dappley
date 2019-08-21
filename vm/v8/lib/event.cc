#include "event.h"
#include "../engine.h"

static FuncTriggerEvent sTriggerEvent = NULL;

void InitializeEvent(FuncTriggerEvent triggerEvent) { sTriggerEvent = triggerEvent; }

void NewEventInstance(Isolate *isolate, Local<Context> context, void *address) {
    Local<ObjectTemplate> eventTpl = ObjectTemplate::New(isolate);
    eventTpl->SetInternalFieldCount(1);

    eventTpl->Set(String::NewFromUtf8(isolate, "trigger"), FunctionTemplate::New(isolate, triggerEventCallback),
                  static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    Local<Object> instance = eventTpl->NewInstance(context).ToLocalChecked();
    instance->SetInternalField(0, External::New(isolate, address));
    context->Global()->DefineOwnProperty(context, String::NewFromUtf8(isolate, "event"), instance,
                                         static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));
}

// storageSetCallback
void triggerEventCallback(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();
    Local<Object> thisArg = info.Holder();
    Local<External> handler = Local<External>::Cast(thisArg->GetInternalField(0));

    if (info.Length() != 2) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "Storage.Set requires 2 arguments"));
        return;
    }

    Local<Value> topic = info[0];
    if (!topic->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "topic must be string"));
        return;
    }

    Local<Value> data = info[1];
    if (!data->IsString()) {
        isolate->ThrowException(String::NewFromUtf8(isolate, "data must be string"));
        return;
    }

    int ret = sTriggerEvent(handler->Value(), *String::Utf8Value(isolate, topic), *String::Utf8Value(isolate, data));

    info.GetReturnValue().Set(ret);
}
