#ifndef __TRANSACTION_STRUCT_H__
#define __TRANSACTION_STRUCT_H__

#include <stdlib.h>

typedef struct {
    char* txid;
    int   vout;
    char* signature;
    char* pubkey;
} transaction_vin_t;

typedef struct {
    long long amount;
    char* pukeyhash; 
} transaction_vout_t;

typedef struct {
    char*  id;
    int    vin_length;
    transaction_vin_t* vin;
    int    vout_length;
    transaction_vout_t* vout;
    unsigned long long tip; 
} transaction_t;

#endif /* __TRANSACTION_STRUCT_H__ */