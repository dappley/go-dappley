#ifndef __TRANSACTION_H__
#define __TRANSACTION_H__ 

#include <v8.h>
using namespace v8;

void NewTransactionInstance(Isolate *isolate, Local<Context> context, void* address);

#endif /* __TRANSACTION_H__ */