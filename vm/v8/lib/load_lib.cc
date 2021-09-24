#include "load_lib.h"
#include "file.h"
#include "memory.h"

void LoadLibraries(Isolate *isolate, Local<Context> &context) {
    LoadBlockchainLibrary(isolate, context);
    LoadStorageLibrary(isolate, context);
    LoadSenderLibrary(isolate, context);
    LoadVerificationLibrary(isolate, context);
}

void LoadBlockchainLibrary(Isolate *isolate, Local<Context> &context){
    LoadLibrary(isolate, context, "jslib/blockchain.js", "blockchain.js");
}

void LoadStorageLibrary(Isolate *isolate, Local<Context> &context){
    LoadLibrary(isolate, context, "jslib/storage.js", "storage.js");
}

void LoadSenderLibrary(Isolate *isolate, Local<Context> &context){
    LoadLibrary(isolate, context, "jslib/sender.js", "sender.js");
}

void LoadVerificationLibrary(Isolate *isolate, Local<Context> &context){
    LoadLibrary(isolate, context, "jslib/verification.js", "verification.js");
}

void LoadLibrary(Isolate *isolate, Local<Context> &context, const char *filepath, const char *filename){
    char *source = readFile(filepath, NULL);
    Local<String> v8source = String::NewFromUtf8(isolate, source, NewStringType::kNormal).ToLocalChecked();
    MyFree(source);
    ScriptOrigin sourceSrcOrigin(String::NewFromUtf8(isolate, filename));
    MaybeLocal<Script> script = Script::Compile(context, v8source,&sourceSrcOrigin);
    script.ToLocalChecked()->Run(context);
}