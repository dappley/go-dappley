// Copyright 2015 the V8 project authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
#include "engine.h"
#include <assert.h>
#include <libplatform/libplatform.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <v8.h>
#include "engine_int.h"
#include "lib/allocator.h"
#include "lib/blockchain.h"
#include "lib/crypto.h"
#include "lib/event.h"
#include "lib/execution_env.h"
#include "lib/global.h"
#include "lib/instruction_counter.h"
#include "lib/load_lib.h"
#include "lib/logger.h"
#include "lib/math.h"
#include "lib/memory.h"
#include "lib/prev_utxo.h"
#include "lib/reward_distributor.h"
#include "lib/storage.h"
#include "lib/transaction.h"
#include "lib/vm_error.h"

using namespace v8;
std::unique_ptr<Platform> platformPtr;

#define ExecuteTimeOut 5 * 1000 * 1000
void EngineLimitsCheckDelegate(Isolate *isolate, size_t count, void *listenerContext);
void PrintException(Isolate *isolate, Local<Context> context, TryCatch &trycatch);
void PrintAndReturnException(Isolate *isolate, char **exception, Local<Context> context, TryCatch &trycatch);

void Initialize() {
    // Initialize V8.
    platformPtr = platform::NewDefaultPlatform();
    V8::InitializePlatform(platformPtr.get());
    V8::Initialize();

    // Initialize V8Engine.
    SetInstructionCounterIncrListener(EngineLimitsCheckDelegate);
}

const char *toCString(const v8::String::Utf8Value &value) { return *value ? *value : "<string conversion failed>"; }

void reportException(v8::Isolate *isolate, v8::TryCatch *try_catch) {
    v8::HandleScope handle_scope(isolate);
    v8::String::Utf8Value exception(isolate, try_catch->Exception());
    const char *exception_string = toCString(exception);
    v8::Local<v8::Message> message = try_catch->Message();
    if (message.IsEmpty()) {
        // V8 didn't provide any extra information about this error; just
        // print the exception.
        fprintf(stderr, "%s\n", exception_string);
    } else {
        // Print (filename):(line number): (message).
        v8::String::Utf8Value filename(isolate, message->GetScriptOrigin().ResourceName());
        v8::Local<v8::Context> context(isolate->GetCurrentContext());
        const char *filename_string = toCString(filename);
        int linenum = message->GetLineNumber(context).FromJust();
        fprintf(stderr, "%s:%i: %s\n", filename_string, linenum, exception_string);
        // Print line of source code.
        v8::String::Utf8Value sourceline(isolate, message->GetSourceLine(context).ToLocalChecked());
        const char *sourceline_string = toCString(sourceline);
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
        if (try_catch->StackTrace(context).ToLocal(&stack_trace_string) && stack_trace_string->IsString() &&
            v8::Local<v8::String>::Cast(stack_trace_string)->Length() > 0) {
            v8::String::Utf8Value stack_trace(isolate, stack_trace_string);
            const char *stack_trace_string = toCString(stack_trace);
            fprintf(stderr, "%s\n", stack_trace_string);
        }
    }
}

int ExecuteSourceDataDelegate(char **result, Isolate *isolate, const char *source, int source_line_offset, Local<Context> context, TryCatch &trycatch,
                              void *delegateContext) {
    // Create a string containing the JavaScript source code.
    Local<String> src = String::NewFromUtf8(isolate, source, NewStringType::kNormal).ToLocalChecked();

    // Compile the source code.
    ScriptOrigin sourceSrcOrigin(String::NewFromUtf8(isolate, "_contract_runner.js"), Integer::New(isolate, source_line_offset));
    MaybeLocal<Script> script = Script::Compile(context, src, &sourceSrcOrigin);

    if (script.IsEmpty()) {
        PrintAndReturnException(isolate, result, context, trycatch);
        return VM_EXCEPTION_ERR;
    }
    // Run the script to get the result.
    MaybeLocal<Value> ret = script.ToLocalChecked()->Run(context);
    if (ret.IsEmpty()) {
        PrintAndReturnException(isolate, result, context, trycatch);
        return VM_EXCEPTION_ERR;
    }
    // set result.
    if (result != NULL) {
        Local<Object> obj = ret.ToLocalChecked().As<Object>();
        if (!obj->IsUndefined()) {
            String::Utf8Value str(isolate, obj);
            *result = (char *)malloc(str.length() + 1);
            strcpy(*result, *str);
        }
    }

    return VM_SUCCESS;
}

int executeV8Script(const char *sourceCode, int source_line_offset, uintptr_t handler, char **result, V8Engine *e) {
    return RunV8ScriptThread(result, e, sourceCode, source_line_offset, handler);
}

// Execute js codes by v8 engine
int ExecuteByV8(const char *sourceCode, int source_line_offset, uintptr_t handler, char **result, V8Engine *e, ExecutionDelegate delegate, void *delegateContext) {
    // Get current Isolate
    Isolate *isolate = static_cast<Isolate *>(e->isolate);
    Locker locker(isolate);

    Isolate::Scope isolate_scope(isolate);

    // Create a stack-allocated handle scope.
    HandleScope handle_scope(isolate);
    Local<ObjectTemplate> globalTpl = CreateGlobalObjectTemplate(isolate);

    // Set up an exception handler
    TryCatch try_catch(isolate);

    // Create a new context.
    Local<Context> context = v8::Context::New(isolate, NULL, globalTpl);

    // Enter the context for compiling and running the hello world script.
    Context::Scope context_scope(context);

    // Continue put objects to global object.
    SetGlobalObjectProperties(isolate, context, e, (void *)handler);

    // Setup execution env.
    if (SetupExecutionEnv(isolate, context)) {
        PrintAndReturnException(isolate, result, context, try_catch);
        return VM_EXCEPTION_ERR;
    }

    LoadLibraries(isolate, context);

    int retTmp = delegate(result, isolate, sourceCode, source_line_offset, context, try_catch, delegateContext);

    if (e->is_unexpected_error_happen) {
        return VM_UNEXPECTED_ERR;
    }

    return retTmp;
}

int CheckContractSyntax(const char* sourceCode, V8Engine *e)
{
  Isolate *isolate = static_cast<Isolate *>(e->isolate);
  Locker locker(isolate);
  int errorCode = 0;
  {
      Isolate::Scope isolate_scope(isolate);

      // Create a stack-allocated handle scope.
      HandleScope handle_scope(isolate);

      // Set up an exception handler
      TryCatch try_catch(isolate);

      // Create a new context.
      Local<Context> context = v8::Context::New(isolate);
      v8::Context::Scope context_scope(context);

      Local<String> source = String::NewFromUtf8(
          isolate,
          sourceCode,
          NewStringType::kNormal
        ).ToLocalChecked();

        // Compile the source code.
      Local<Script> script;
      if (!Script::Compile(context, source).ToLocal(&script)) {
          reportException(isolate, &try_catch);
          errorCode = 1;
          script.Clear();
      }
  }
  return errorCode;
}

void DisposeV8() {
    V8::Dispose();
    V8::ShutdownPlatform();
    if (platformPtr) {
        platformPtr = NULL;
    }
}

V8Engine *CreateEngine() {
    ArrayBuffer::Allocator *allocator = new ArrayBufferAllocator();

    Isolate::CreateParams create_params;
    create_params.array_buffer_allocator = allocator;

    Isolate *isolate = Isolate::New(create_params);

    V8Engine *e = (V8Engine *)calloc(1, sizeof(V8Engine));
    e->allocator = allocator;
    e->isolate = isolate;
    e->timeout = ExecuteTimeOut;
    e->ver = BUILD_DEFAULT_VER;  // default load initial com
    return e;
}

void ReadMemoryStatistics(V8Engine *e) {
    Isolate *isolate = static_cast<Isolate *>(e->isolate);
    ArrayBufferAllocator *allocator = static_cast<ArrayBufferAllocator *>(e->allocator);

    HeapStatistics heap_stats;
    isolate->GetHeapStatistics(&heap_stats);

    V8EngineStats *stats = &(e->stats);
    stats->heap_size_limit = heap_stats.heap_size_limit();
    stats->malloced_memory = heap_stats.malloced_memory();
    stats->peak_malloced_memory = heap_stats.peak_malloced_memory();
    stats->total_available_size = heap_stats.total_available_size();
    stats->total_heap_size = heap_stats.total_heap_size();
    stats->total_heap_size_executable = heap_stats.total_heap_size_executable();
    stats->total_physical_size = heap_stats.total_physical_size();
    stats->used_heap_size = heap_stats.used_heap_size();
    stats->total_array_buffer_size = allocator->total_available_size();
    stats->peak_array_buffer_size = allocator->peak_allocated_size();

    stats->total_memory_size = stats->total_heap_size + stats->peak_array_buffer_size;
}

void TerminateExecution(V8Engine *e) {
    if (e->is_requested_terminate_execution) {
        return;
    }
    Isolate *isolate = static_cast<Isolate *>(e->isolate);
    isolate->TerminateExecution();
    e->is_requested_terminate_execution = true;
}

void SetInnerContractErrFlag(V8Engine *e) { e->is_inner_vm_error_happen = true; }

void DeleteEngine(V8Engine *e) {
    Isolate *isolate = static_cast<Isolate *>(e->isolate);
    isolate->Dispose();

    delete static_cast<ArrayBuffer::Allocator *>(e->allocator);

    free(e);
}

int IsEngineLimitsExceeded(V8Engine *e) {
    // TODO: read memory stats everytime may impact the performance.
    ReadMemoryStatistics(e);
    if (e->limits_of_executed_instructions > 0 && e->limits_of_executed_instructions < e->stats.count_of_executed_instructions) {
        // Reach instruction limits.
        return VM_GAS_LIMIT_ERR;
    } else if (e->limits_of_total_memory_size > 0 && e->limits_of_total_memory_size < e->stats.total_memory_size) {
        // reach memory limits.
        return VM_MEM_LIMIT_ERR;
    }
    return 0;
}

void EngineLimitsCheckDelegate(Isolate *isolate, size_t count, void *listenerContext) {
    V8Engine *e = static_cast<V8Engine *>(listenerContext);

    if (IsEngineLimitsExceeded(e)) {
        TerminateExecution(e);
    }
}

void PrintException(Isolate *isolate, Local<Context> context, TryCatch &trycatch) { PrintAndReturnException(isolate, NULL, context, trycatch); }

void PrintAndReturnException(Isolate *isolate, char **exception, Local<Context> context, TryCatch &trycatch) {
    static char SOURCE_INFO_PLACEHOLDER[] = "";
    char *source_info = NULL;

    // print source line.
    Local<Message> message = trycatch.Message();
    if (!message.IsEmpty()) {
        // Print (filename):(line number): (message).
        ScriptOrigin origin = message->GetScriptOrigin();
        String::Utf8Value filename(isolate, message->GetScriptResourceName());
        int linenum = message->GetLineNumber(context).FromMaybe(0);
        // Print line of source code.
        String::Utf8Value sourceline(isolate, message->GetSourceLine(context).ToLocalChecked());
        int script_start = (linenum - origin.ResourceLineOffset()->Value()) == 1 ? origin.ResourceColumnOffset()->Value() : 0;
        int start = message->GetStartColumn(context).FromMaybe(0);
        int end = message->GetEndColumn(context).FromMaybe(0);
        if (start >= script_start) {
            start -= script_start;
            end -= script_start;
        }
        char arrow[start + 2];
        for (int i = 0; i < start; i++) {
            char c = (*sourceline)[i];
            if (c == '\t') {
                arrow[i] = c;
            } else {
                arrow[i] = ' ';
            }
        }
        arrow[start] = '^';
        arrow[start + 1] = '\0';

        asprintf(&source_info, "%s:%d\n%s\n%s\n", *filename, linenum, *sourceline, arrow);
    }

    if (source_info == NULL) {
        source_info = SOURCE_INFO_PLACEHOLDER;
    }

    // get stack trace.
    MaybeLocal<Value> stacktrace_ret = trycatch.StackTrace(context);
    if (!stacktrace_ret.IsEmpty()) {
        // print full stack trace.
        String::Utf8Value stack_str(isolate, stacktrace_ret.ToLocalChecked());
        printf("V8 Exception:\n%s%s", source_info, *stack_str);
    }

    // exception message.
    Local<Value> exceptionValue = trycatch.Exception();
    String::Utf8Value exception_str(isolate, exceptionValue);
    if (stacktrace_ret.IsEmpty()) {
        // print exception when stack trace is not available.
        printf("V8 Exception:\n%s%s", source_info, *exception_str);
    }

    if (source_info != NULL && source_info != SOURCE_INFO_PLACEHOLDER) {
        free(source_info);
    }

    // return exception message.
    if (exception != NULL) {
        *exception = (char *)malloc(exception_str.length() + 1);
        if (exception_str.length() == 0) {
            strcpy(*exception, "");
        } else {
            strcpy(*exception, *exception_str);
        }
    }
}