#include "logger.h"
#include "../engine.h"

static FuncLogger sLogger = NULL;

static void LogCallback(unsigned int level, const FunctionCallbackInfo<Value> &info);

void InitializeLogger(FuncLogger logger) {
    sLogger = logger;
}

void NewLoggerInstance(Isolate *isolate, Local<Context> context, void* address) {
    Local<ObjectTemplate> loggerTpl = ObjectTemplate::New(isolate);
    
    loggerTpl->Set(String::NewFromUtf8(isolate, "debug"),
                FunctionTemplate::New(isolate, LogDebugCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

    loggerTpl->Set(String::NewFromUtf8(isolate, "info"),
                FunctionTemplate::New(isolate, LogInfoCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

    loggerTpl->Set(String::NewFromUtf8(isolate, "warn"),
                FunctionTemplate::New(isolate, LogWarnCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

    loggerTpl->Set(String::NewFromUtf8(isolate, "error"),
                FunctionTemplate::New(isolate, LogErrorCallback),
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                               PropertyAttribute::ReadOnly));

    Local<Object> instance = loggerTpl->NewInstance(context).ToLocalChecked();
    context->Global()->DefineOwnProperty(
                context, String::NewFromUtf8(isolate, "_log"),
                instance,
                static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                     PropertyAttribute::ReadOnly));
}

void LogCallback(unsigned int level, const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();
    Local<Object> thisArg = info.Holder();

    char** args = (char **)malloc(sizeof(char *) * info.Length());

    for (int i = 0; i < info.Length(); i++) {
        args[i] = *String::Utf8Value(isolate, info[i]);
    }

    sLogger(level, args, info.Length());
}

void LogDebugCallback(const FunctionCallbackInfo<Value> &info){
    LogCallback(0, info);
}

void LogInfoCallback(const FunctionCallbackInfo<Value> &info) {
    LogCallback(1, info);
}

void LogWarnCallback(const FunctionCallbackInfo<Value> &info) {
    LogCallback(2, info);
}

void LogErrorCallback(const FunctionCallbackInfo<Value> &info) {
    LogCallback(3, info);
}
