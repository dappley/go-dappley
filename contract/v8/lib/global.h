#ifndef _DAPPLEY_NF_VM_V8_LIB_GLOBAL_H_
#define _DAPPLEY_NF_VM_V8_LIB_GLOBAL_H_

#include <v8.h>
#include "../engine.h"

using namespace v8;

Local<ObjectTemplate> CreateGlobalObjectTemplate(Isolate *isolate);

void SetGlobalObjectProperties(Isolate *isolate, Local<Context> context, V8Engine *e, void *handler);

void *GetV8EngineHandler(Local<Context> context);

#endif
