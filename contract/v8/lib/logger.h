#ifndef __LOGGER_H__
#define __LOGGER_H__

#include "v8.h"
using namespace v8;

void NewLoggerInstance(Isolate *isolate, Local<Context> context, void* address);

void LogDebugCallback(const FunctionCallbackInfo<Value> &info);
void LogInfoCallback(const FunctionCallbackInfo<Value> &info);
void LogWarnCallback(const FunctionCallbackInfo<Value> &info);
void LogErrorCallback(const FunctionCallbackInfo<Value> &info);

#endif /* __LOGGER_H__ */