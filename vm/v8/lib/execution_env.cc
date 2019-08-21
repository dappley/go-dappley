#include "execution_env.h"
#include "../engine.h"
#include "file.h"
#include "global.h"
#include "logger.h"
#include "string.h"
static AttachLibVersionDelegate alvDelegate = NULL;

int SetupExecutionEnv(Isolate *isolate, Local<Context> &context) {
    char *verlib = NULL;
    if (alvDelegate != NULL) {
        void *handler = GetV8EngineHandler(context);
        verlib = alvDelegate((void *)handler, "execution_env.js");
    }
    if (verlib == NULL) {
        return 1;
    }

    char path[64] = {0};
    strcat(path, verlib);
    free(verlib);
    char *data = readFile(path, NULL);
    // char *data = readFile("lib/execution_env.js", NULL);
    if (data == NULL) {
        isolate->ThrowException(Exception::Error(String::NewFromUtf8(isolate, "execution_env.js is not found.")));
        return 1;
    }

    Local<String> source = String::NewFromUtf8(isolate, data, NewStringType::kNormal).ToLocalChecked();
    free(data);

    // Compile the source code.
    ScriptOrigin sourceSrcOrigin(String::NewFromUtf8(isolate, "execution_env.js"));
    MaybeLocal<Script> script = Script::Compile(context, source, &sourceSrcOrigin);

    if (script.IsEmpty()) {
        return 1;
    }

    // Run the script to get the result.
    MaybeLocal<Value> v = script.ToLocalChecked()->Run(context);
    if (v.IsEmpty()) {
        return 1;
    }
    return 0;
}

void InitializeExecutionEnvDelegate(AttachLibVersionDelegate aDelegate) { alvDelegate = aDelegate; }
