#ifndef __UTIL_H__
#define __UTIL_H__

#include <v8.h>

using namespace v8;
v8::Local<v8::BigInt> CastStringToBigInt(v8::Isolate *isolate,const char* s);

#endif /* __UTIL_H__ */