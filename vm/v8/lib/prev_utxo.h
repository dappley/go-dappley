#ifndef __PREV_UTXO_H__
#define __PREV_UTXO_H__

#include <v8.h>
using namespace v8;

void NewPrevUtxoInstance(Isolate* isolate, Local<Context> context, void* address);

#endif /* __PREV_UTXO_H__ */