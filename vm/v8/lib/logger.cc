#include "logger.h"
#include "../engine.h"
#include "memory.h"

static FuncLogger sLogger = NULL;

static void LogCallback(unsigned int level, const FunctionCallbackInfo<Value> &info);

void InitializeLogger(FuncLogger logger) { sLogger = logger; }

void NewLoggerInstance(Isolate *isolate, Local<Context> context, void *address) {
    Local<ObjectTemplate> loggerTpl = ObjectTemplate::New(isolate);

    loggerTpl->Set(String::NewFromUtf8(isolate, "debug"), FunctionTemplate::New(isolate, LogDebugCallback),
                   static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    loggerTpl->Set(String::NewFromUtf8(isolate, "info"), FunctionTemplate::New(isolate, LogInfoCallback),
                   static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    loggerTpl->Set(String::NewFromUtf8(isolate, "warn"), FunctionTemplate::New(isolate, LogWarnCallback),
                   static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    loggerTpl->Set(String::NewFromUtf8(isolate, "error"), FunctionTemplate::New(isolate, LogErrorCallback),
                   static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));

    Local<Object> instance = loggerTpl->NewInstance(context).ToLocalChecked();
    context->Global()->DefineOwnProperty(context, String::NewFromUtf8(isolate, "_log"), instance,
                                         static_cast<PropertyAttribute>(PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly));
}

void LogCallback(unsigned int level, const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();

    char **args = (char **)MyMalloc(sizeof(char *) * info.Length());
    String::Utf8Value **utf8Values = new String::Utf8Value *[info.Length()];
    for (int i = 0; i < info.Length(); i++) {
        String::Utf8Value *str = new String::Utf8Value(isolate, info[i]);
        args[i] = **str;
        utf8Values[i] = str;
    }

    sLogger(level, args, info.Length());
    for (int i = 0; i < info.Length(); i++) {
        delete utf8Values[i];
    }
    delete[] utf8Values;
    MyFree(args);
}

void LogDebugCallback(const FunctionCallbackInfo<Value> &info) { LogCallback(0, info); }

void LogInfoCallback(const FunctionCallbackInfo<Value> &info) { LogCallback(1, info); }

void LogWarnCallback(const FunctionCallbackInfo<Value> &info) { LogCallback(2, info); }

void LogErrorCallback(const FunctionCallbackInfo<Value> &info) { LogCallback(3, info); }
