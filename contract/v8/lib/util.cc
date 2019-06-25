#include "util.h"
v8::Local<v8::BigInt> CastStringToBigInt(v8::Local<v8::Context> *context,v8::Isolate *isolate,const char* s)
{
    v8::Local<v8::String> vStr=v8::String::NewFromUtf8(isolate, s);
    MaybeLocal<v8::BigInt> mBi=vStr->ToBigInt(*context);
    v8::Local<v8::BigInt> vBigInt=mBi.ToLocalChecked();
    return vBigInt;
}