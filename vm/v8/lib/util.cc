#include "util.h"
v8::Local<v8::BigInt> CastStringToBigInt(v8::Local<v8::Context> *context,v8::Isolate *isolate,const char* s)
{
    v8::Local<v8::String> vStr=v8::String::NewFromUtf8(isolate, s);
    MaybeLocal<v8::BigInt> mBi=vStr->ToBigInt(*context);
    v8::Local<v8::BigInt> vBigInt=mBi.ToLocalChecked();
    return vBigInt;
}

std::string ReplaceAll(std::string str, const std::string &from, const std::string &to) {
    size_t from_len = from.length(), to_len = to.length();
    size_t start_pos = 0;
    while ((start_pos = str.find(from, start_pos)) != std::string::npos) {
        str.replace(start_pos, from_len, to);
        start_pos += to_len;
    }
    return str;
}