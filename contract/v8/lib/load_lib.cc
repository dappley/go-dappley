#include "load_lib.h"
#include "file.h"

void LoadLibraries(Isolate *isolate, Local<Context> &context){

    char *source = readFile("jslib/blockchain.js", NULL);
    Local<String> v8source = String::NewFromUtf8(isolate, source, NewStringType::kNormal).ToLocalChecked();
    ScriptOrigin sourceSrcOrigin(String::NewFromUtf8(isolate, "blockchain.js"));
    MaybeLocal<Script> script = Script::Compile(context, v8source,&sourceSrcOrigin);
    script.ToLocalChecked()->Run(context);
}
