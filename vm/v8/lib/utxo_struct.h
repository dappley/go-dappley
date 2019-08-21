#ifndef __UTXO_STRUCT_H__
#define __UTXO_STRUCT_H__

#include <stdio.h>

struct utxo_t {
    char* txid;
    int   tx_index;
    char* value;
    char* pubkeyhash;
    char* address;
};

#endif /* __UTXO_STRUCT_H__ */