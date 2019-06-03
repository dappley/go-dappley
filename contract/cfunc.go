package vm

/*
#include <stdbool.h>
#include "v8/lib/transaction_struct.h"
// require.
char *RequireDelegateFunc(void *handler, const char *filename, size_t *lineOffset);
char *AttachLibVersionDelegateFunc(void *handler, const char *libname);

bool  VerifyAddressFunc(const char* address, size_t *gasCnt);
int   TransferFunc(void *handler, const char *to, const char *amount, const char *tip, size_t *gasCnt);
int   GetCurrBlockHeightFunc(void *handler);
char* GetNodeAddressFunc(void *handler);

char* StorageGetFunc(void *address, const char *key);
int   StorageSetFunc(void *address,const char *key, const char *value);
int   StorageDelFunc(void *address,const char *key);
int   TriggerEventFunc(void *address, const char *topic, const char *data);
void  TransactionGetFunc(void *address, void *context);
void  LoggerFunc(unsigned int level, char ** args, int length);
int	  RecordRewardFunc(void *handler, const char *address, const char *amount);
void  PrevUtxoGetFunc(void *address, void* context);
bool  VerifySignatureFunc(const char *msg, const char *pubkey, const char *sig);
bool  VerifyPublicKeyFunc(const char *addr, const char *pubkey);
int RandomFunc(void *handler, int max);

void* Malloc(size_t size);
void  Free(void* address);

char *Cgo_RequireDelegateFunc(void *handler, const char *filename, size_t *lineOffset) {
	return RequireDelegateFunc(handler, filename, lineOffset);
}

char *Cgo_AttachLibVersionDelegateFunc(void *handler, const char *libname) {
	return AttachLibVersionDelegateFunc(handler, libname);
}

bool Cgo_VerifyAddressFunc(const char *address, size_t *gasCnt) {
	return VerifyAddressFunc(address, gasCnt);
};

int Cgo_TransferFunc(void *handler, const char *to, const char *amount, const char *tip, size_t *gasCnt) {
	return TransferFunc(handler, to, amount, tip, gasCnt);
};

int Cgo_GetCurrBlockHeightFunc(void *handler){
	return GetCurrBlockHeightFunc(handler);
};

char* Cgo_GetNodeAddressFunc(void *handler){
	return GetNodeAddressFunc(handler);
};

char* Cgo_StorageGetFunc(void *address, const char *key){
	return StorageGetFunc(address,key);
};

int Cgo_StorageSetFunc(void *address, const char *key, const char *value){
	return StorageSetFunc(address,key, value);
};

int Cgo_StorageDelFunc(void *address, const char *key){
	return StorageDelFunc(address,key);
};

int Cgo_TriggerEventFunc(void *address, const char *topic, const char *data){
	return TriggerEventFunc(address, topic, data);
};

void Cgo_TransactionGetFunc(void *address, void *context) {
	TransactionGetFunc(address, context);
}

void Cgo_LoggerFunc(unsigned int level, char ** args, int length) {
	return LoggerFunc(level, args, length);
}

int	Cgo_RecordRewardFunc(void *handler, const char *address, const char *amount){
	return RecordRewardFunc(handler, address,amount);
}

void  Cgo_PrevUtxoGetFunc(void *address, void* context) {
	return PrevUtxoGetFunc(address, context);
}

int Cgo_RandomFunc(void *handler, int max){
	return RandomFunc(handler, max);
}

bool Cgo_VerifySignatureFunc(const char *msg, const char *pubkey, const char *sig){
	return VerifySignatureFunc(msg, pubkey, sig);
}

bool Cgo_VerifyPublicKeyFunc(const char *addr, const char *pubkey){
	return VerifyPublicKeyFunc(addr, pubkey);
}

void* Cgo_Malloc(size_t size) {
    return Malloc(size);
}

void  Cgo_Free(void* address) {
	Free(address);
}


*/
import "C"
