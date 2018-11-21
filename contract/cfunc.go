package sc

/*
#include <stdbool.h>
#include "v8/lib/transaction_struct.h"

bool  VerifyAddressFunc(const char* address);

char* StorageGetFunc(void *address, const char *key);
int   StorageSetFunc(void *address,const char *key, const char *value);
int   StorageDelFunc(void *address,const char *key);
void TransactionGetFunc(void *address, void *context);
void LoggerFunc(unsigned int level, char ** args, int length);

bool Cgo_VerifyAddressFunc(const char *address) {
	return VerifyAddressFunc(address);
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

void Cgo_TransactionGetFunc(void *address, void *context) {
	TransactionGetFunc(address, context);
}

void Cgo_LoggerFunc(unsigned int level, char ** args, int length) {
	return LoggerFunc(level, args, length);
}

*/
import "C"
