#ifndef __UTIL_H__
#define __UTIL_H__

#include <v8.h>
#include <string>

using namespace v8;
v8::Local<v8::BigInt> CastStringToBigInt(v8::Local<v8::Context> *context,v8::Isolate *isolate,const char* s);

std::string ReplaceAll(std::string str, const std::string &from, const std::string &to);
#endif /* __UTIL_H__ */