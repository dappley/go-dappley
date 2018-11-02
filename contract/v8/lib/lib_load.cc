#include "lib_load.h"
#include "file.h"

void LoadLibraries(Isolate *isolate, Local<Context> &context){

    char *data = readFile("jslib/blockchain.js", NULL);
    Local<String> source = String::NewFromUtf8(isolate, data, NewStringType::kNormal).ToLocalChecked();
    ScriptOrigin sourceSrcOrigin(String::NewFromUtf8(isolate, "blockchain.js"));
    MaybeLocal<Script> script = Script::Compile(context, source,&sourceSrcOrigin);
    script.ToLocalChecked()->Run(context);
}