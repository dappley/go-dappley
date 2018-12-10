// Copyright 2015 the V8 project authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include <v8.h>
#include <libplatform/libplatform.h>
#include "engine.h"
#include "lib/blockchain.h"
#include "lib/load_lib.h"
#include "lib/load_sc.h"
#include "lib/storage.h"
#include "lib/logger.h"
#include "lib/transaction.h"
#include "lib/reward_distributor.h"
#include "lib/prev_utxo.h"
#include "lib/crypto.h"
#include "lib/math.h"

using namespace v8;
std::unique_ptr<Platform> platformPtr;

void Initialize(){
    // Initialize V8.
    platformPtr = platform::NewDefaultPlatform();
    V8::InitializePlatform(platformPtr.get());
    V8::Initialize();
}

const char* toCString(const v8::String::Utf8Value& value) {
  return *value ? *value : "<string conversion failed>";
}

void reportException(v8::Isolate* isolate, v8::TryCatch* try_catch) {
  v8::HandleScope handle_scope(isolate);
  v8::String::Utf8Value exception(isolate, try_catch->Exception());
  const char* exception_string = toCString(exception);
  v8::Local<v8::Message> message = try_catch->Message();
  if (message.IsEmpty()) {
    // V8 didn't provide any extra information about this error; just
    // print the exception.
    fprintf(stderr, "%s\n", exception_string);
  } else {
    // Print (filename):(line number): (message).
    v8::String::Utf8Value filename(isolate,
      message->GetScriptOrigin().ResourceName());
    v8::Local<v8::Context> context(isolate->GetCurrentContext());
    const char* filename_string = toCString(filename);
    int linenum = message->GetLineNumber(context).FromJust();
    fprintf(stderr, "%s:%i: %s\n", filename_string, linenum, exception_string);
    // Print line of source code.
    v8::String::Utf8Value sourceline(
      isolate, message->GetSourceLine(context).ToLocalChecked());
    const char* sourceline_string = toCString(sourceline);
    fprintf(stderr, "%s\n", sourceline_string);
    // Print wavy underline (GetUnderline is deprecated).
    int start = message->GetStartColumn(context).FromJust();
    for (int i = 0; i < start; i++) {
      fprintf(stderr, " ");
    }
    int end = message->GetEndColumn(context).FromJust();
    for (int i = start; i < end; i++) {
      fprintf(stderr, "^");
    }
    fprintf(stderr, "\n");
    v8::Local<v8::Value> stack_trace_string;
    if (try_catch->StackTrace(context).ToLocal(&stack_trace_string) &&
    stack_trace_string->IsString() &&
    v8::Local<v8::String>::Cast(stack_trace_string)->Length() > 0) {
      v8::String::Utf8Value stack_trace(isolate, stack_trace_string);
      const char* stack_trace_string = toCString(stack_trace);
      fprintf(stderr, "%s\n", stack_trace_string);
    }
  }
}

int executeV8Script(const char *sourceCode, uintptr_t handler, char **result) {

  // Create a new Isolate and make it the current one.
  Isolate::CreateParams create_params;
  create_params.array_buffer_allocator = ArrayBuffer::Allocator::NewDefaultAllocator();
  Isolate* isolate = Isolate::New(create_params);
  int errorCode = 0;

  {
    Isolate::Scope isolate_scope(isolate);

    // Create a stack-allocated handle scope.
    HandleScope handle_scope(isolate);
    //
    Local<ObjectTemplate> globalTpl = NewNativeRequireFunction(isolate);

    // Set up an exception handler
    TryCatch try_catch(isolate);

    // Create a new context.
    Local<Context> context = v8::Context::New(isolate, NULL, globalTpl);

    // Enter the context for compiling and running the hello world script.
    Context::Scope context_scope(context);

    NewBlockchainInstance(isolate, context, (void *)handler);
    NewCryptoInstance(isolate, context, (void *)handler);
    NewStorageInstance(isolate, context, (void *)handler);
    NewLoggerInstance(isolate, context, (void *)handler);
    NewTransactionInstance(isolate, context, (void *)handler);
    NewRewardDistributorInstance(isolate, context, (void *)handler);
    NewPrevUtxoInstance(isolate, context, (void *)handler);
    NewMathInstance(isolate, context, (void *)handler);

    LoadLibraries(isolate, context);
    {

      // Create a string containing the JavaScript source code.
      Local<String> source = String::NewFromUtf8(
        isolate,
        sourceCode,
        NewStringType::kNormal
      ).ToLocalChecked();

      // Compile the source code.
      Local<Script> script;
      if (!Script::Compile(context, source).ToLocal(&script)) {
        reportException(isolate, &try_catch);
        *result = strdup("1");
        errorCode = 1;
        goto RET;
      }

      // Run the script to get the result.
      Local<Value> scriptRes;
      if (!script->Run(context).ToLocal(&scriptRes)) {
        assert(try_catch.HasCaught());
        reportException(isolate, &try_catch);
        *result = strdup("1");
        errorCode = 1;
        goto RET;
      }

      // set result.
      if (result != NULL)  {
        Local<Object> obj = scriptRes.As<Object>();
        if (!obj->IsUndefined()) {
          String::Utf8Value str(isolate, obj);
          *result = (char *)malloc(str.length() + 1);
          strcpy(*result, *str);
        }
      }
    }
  }

RET:
  // Dispose the isolate and tear down V8.
  isolate->Dispose();

  delete create_params.array_buffer_allocator;
  return errorCode;
}

void DisposeV8(){
  V8::Dispose();
  V8::ShutdownPlatform();
}

void V8Free(void *data) {
  free(data);
}