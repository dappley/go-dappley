#include <v8.h>

using namespace v8;

void LoadLibraries(Isolate *isolate, Local<Context> &context);
void LoadBlockchainLibrary(Isolate *isolate, Local<Context> &context);
void LoadStorageLibrary(Isolate *isolate, Local<Context> &context);