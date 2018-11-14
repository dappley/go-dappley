package sc


/*
#include <stdbool.h>

bool  VerifyAddressFunc(const char* address);
int  TransferFunc(void *handler, const char *to, const char *amount, const char *tip);

char* StorageGetFunc(void *address, const char *key);
int   StorageSetFunc(void *address,const char *key, const char *value);
int   StorageDelFunc(void *address,const char *key);
int	  RecordRewardFunc(void *handler, const char *address, const char *amount);

bool Cgo_VerifyAddressFunc(const char *address) {
	return VerifyAddressFunc(address);
};

int Cgo_TransferFunc(void *handler, const char *to, const char *amount, const char *tip) {
	return TransferFunc(handler, to, amount, tip);
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

int	Cgo_RecordRewardFunc(void *handler, const char *address, const char *amount){
	return RecordRewardFunc(handler, address,amount);
}

*/
import "C"
