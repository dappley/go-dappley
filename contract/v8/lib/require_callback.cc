#include "require_callback.h"
#include "../engine.h"
#include "file.h"
#include "global.h"
#include "logger.h"

#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <v8.h>

using namespace v8;

static char source_require_format[] =
    "(function(){\n"
    "return function (exports, module, require) {\n"
    "%s\n"
    "};\n"
    "})();\n";

static RequireDelegate sRequireDelegate = NULL;
static AttachLibVersionDelegate attachLibVersionDelegate = NULL;

static int readSource(Local<Context> context, const char *filename, char **data,
                      size_t *lineOffset) {
  if (strstr(filename, "\"") != NULL) {
    return -1;
  }

  *lineOffset = 0;

  char *content = NULL;

  // try sRequireDelegate.
  if (sRequireDelegate != NULL) {
    V8Engine *e = GetV8EngineInstance(context);
    content = sRequireDelegate(e, filename, lineOffset);
  }

  if (content == NULL) {
    size_t file_size = 0;
    content = readFile(filename, &file_size);
    if (content == NULL) {
      return 1;
    }
  }

  asprintf(data, source_require_format, content);
  *lineOffset += -2;
  free(content);

  return 0;
}

static void attachVersion(char *out, int maxoutlen, Local<Context> context, const char *libname) {

  char *verlib = NULL;
  if (attachLibVersionDelegate != NULL) {
    V8Engine *e = GetV8EngineInstance(context);
    verlib = attachLibVersionDelegate(e, libname);
  }
  if (verlib != NULL) {
    strncat(out, verlib, maxoutlen - strlen(out) - 1);
    free(verlib);
  }
}

void NewNativeRequireFunction(Isolate *isolate,
                              Local<ObjectTemplate> globalTpl) {
  globalTpl->Set(String::NewFromUtf8(isolate, "_native_require"),
                 FunctionTemplate::New(isolate, RequireCallback),
                 static_cast<PropertyAttribute>(PropertyAttribute::DontDelete |
                                                PropertyAttribute::ReadOnly));
}

void RequireCallback(const v8::FunctionCallbackInfo<v8::Value> &info) {
  Isolate *isolate = info.GetIsolate();
  Local<Context> context = isolate->GetCurrentContext();

  if (info.Length() == 0) {
    isolate->ThrowException(
        Exception::Error(String::NewFromUtf8(isolate, "require missing path")));
    return;
  }

  Local<Value> path = info[0];
  if (!path->IsString()) {
    isolate->ThrowException(Exception::Error(
        String::NewFromUtf8(isolate, "require path must be string")));
    return;
  }

  String::Utf8Value filename(isolate, path);
  if (filename.length() >= MAX_PATH_LEN) {
    isolate->ThrowException(Exception::Error(
        String::NewFromUtf8(isolate, "require path length more")));
    return;
  }
  char *abPath = NULL;
  if (strcmp(*filename, LIB_WHITE)) { // if needed, check array instead.
    char versionlizedPath[MAX_VERSIONED_PATH_LEN] = {0};
    attachVersion(versionlizedPath, MAX_VERSIONED_PATH_LEN, context, *filename);
    abPath = realpath(versionlizedPath, NULL);
    if (abPath == NULL) {
      isolate->ThrowException(Exception::Error(String::NewFromUtf8(
          isolate, "require path is invalid absolutepath")));
      return;
    }
    char curPath[MAX_VERSIONED_PATH_LEN] = {0};
    if (curPath[0] == 0x00 && !getCurAbsolute(curPath, MAX_VERSIONED_PATH_LEN)) {
      isolate->ThrowException(Exception::Error(
          String::NewFromUtf8(isolate, "invalid cwd absolutepath")));
      free(abPath);
      return;
    }
    int curLen = strlen(curPath);
    if (strncmp(abPath, curPath, curLen) != 0) {
      isolate->ThrowException(Exception::Error(
          String::NewFromUtf8(isolate, "require path is not in lib")));
      free(abPath);
      return;
    } 

    //free(abPath);
    if (!isFile(abPath)) {
      isolate->ThrowException(Exception::Error(
          String::NewFromUtf8(isolate, "require path is not file")));
      free(abPath);
      return;
    }
  }
  char *pFile = abPath;
  if (abPath == NULL) {
    pFile = *filename;
  }
  char *data = NULL;
  size_t lineOffset = 0;
  if (readSource(context, (const char*)pFile, &data, &lineOffset)) {
    char msg[512];
    snprintf(msg, 512, "require cannot find module '%s'", pFile);
    isolate->ThrowException(
        Exception::Error(String::NewFromUtf8(isolate, msg)));
    free(abPath);
    return;
  }
  free(abPath);
if (!strcmp(*filename, LIB_WHITE)) {
printf("RequireCallback read data: %s", data);
}
  ScriptOrigin sourceSrcOrigin(path, Integer::New(isolate, lineOffset));
  MaybeLocal<Script> script = Script::Compile(
      context, String::NewFromUtf8(isolate, data), &sourceSrcOrigin);
  if (!script.IsEmpty()) {
    MaybeLocal<Value> ret = script.ToLocalChecked()->Run(context);
    if (!ret.IsEmpty()) {
      Local<Value> rr = ret.ToLocalChecked();
      info.GetReturnValue().Set(rr);
    }
  }

  free(static_cast<void *>(data));
}

void InitializeRequireDelegate(RequireDelegate delegate, AttachLibVersionDelegate aDelegate) {
  sRequireDelegate = delegate;
  attachLibVersionDelegate = aDelegate;
}
