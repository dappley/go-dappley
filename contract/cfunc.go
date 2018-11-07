package sc


/*
#include <stdbool.h>

bool  VerifyAddressFunc(const char* address);

char* StorageGetFunc(const char *key);
int   StorageSetFunc(const char *key, const char *value);
int   StorageDelFunc(const char *key);

bool Cgo_VerifyAddressFunc(const char *address) {
	return VerifyAddressFunc(address);
};
char* Cgo_StorageGetFunc(const char *key){
	return StorageGetFunc(key);
};

int Cgo_StorageSetFunc(const char *key, const char *value){
	return StorageSetFunc(key, value);
};

int Cgo_StorageDelFunc(const char *key){
	return StorageDelFunc(key);
};
*/
import "C"
