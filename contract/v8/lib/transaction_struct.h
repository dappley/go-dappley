#ifndef __TRANSACTION_STRUCT_H__
#define __TRANSACTION_STRUCT_H__

#include <stdlib.h>

struct transaction_vin_t {
    char* txid;
    int   vout;
    char* signature;
    char* pubkey;
} ;

struct transaction_vout_t {
    long long amount;
    char* pubkeyhash; 
} ;

struct transaction_t {
    char*  id;
    int    vin_length;
    struct transaction_vin_t* vin;
    int    vout_length;
    struct transaction_vout_t* vout;
    unsigned long long tip; 
} ;

#endif /* __TRANSACTION_STRUCT_H__ */
