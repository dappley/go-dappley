#include "load_lib.h"
#include "file.h"
#include "memory.h"

void LoadLibraries(Isolate *isolate, Local<Context> &context) {
    LoadBlockchainLibrary(isolate, context);
    LoadStorageLibrary(isolate, context);
}

void LoadBlockchainLibrary(Isolate *isolate, Local<Context> &context) {
    char *source = readFile("jslib/blockchain.js", NULL);
    Local<String> v8source = String::NewFromUtf8(isolate, source, NewStringType::kNormal).ToLocalChecked();
    MyFree(source);
    ScriptOrigin sourceSrcOrigin(String::NewFromUtf8(isolate, "blockchain.js"));
    MaybeLocal<Script> script = Script::Compile(context, v8source, &sourceSrcOrigin);
    script.ToLocalChecked()->Run(context);
}

void LoadStorageLibrary(Isolate *isolate, Local<Context> &context) {
    char *source = readFile("jslib/storage.js", NULL);
    Local<String> v8source = String::NewFromUtf8(isolate, source, NewStringType::kNormal).ToLocalChecked();
    MyFree(source);
    ScriptOrigin sourceSrcOrigin(String::NewFromUtf8(isolate, "storage.js"));
    MaybeLocal<Script> script = Script::Compile(context, v8source, &sourceSrcOrigin);
    script.ToLocalChecked()->Run(context);
}
