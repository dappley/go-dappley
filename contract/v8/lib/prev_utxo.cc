#include "utxo.h"
#include "../engine.h"

const PropertyAttribute DEFAULT_PROPERTY = static_cast<PropertyAttribute>(
    PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly);

static FuncPrevUtxoGet utxoGet = NULL;
typedef struct {
    Isolate *isolate;
    Local<Context> *context;
} UtxoContext;

void InitializePrevUtxo(FuncPrevUtxoGet get) {
    utxoGet = get;
}

void SetPrevUtxoData(struct utxo_t* utxos, int length,  void* context) {
    if (utxos == NULL || context == NULL) {
        return;
    }

    UtxoContext* utxoContext = static_cast<UtxoContext*>(context);
    Local<Array> utxosInstance = Array::New(utxoContext->isolate, length);
    for (int i = 0; i < length; i++) {
        Local<Object> utxoInstance = Object::New(utxoContext->isolate);
        utxoInstance->DefineOwnProperty(
            *(utxoContext->context),
            String::NewFromUtf8(utxoContext->isolate, "txid"),
            String::NewFromUtf8(utxoContext->isolate, utxos[i].txid),
            DEFAULT_PROPERTY
        );

        utxoInstance->DefineOwnProperty(
            *(utxoContext->context),
            String::NewFromUtf8(utxoContext->isolate, "txIndex"),
            Integer::New(utxoContext->isolate, utxos[i].tx_index),
            DEFAULT_PROPERTY
        );

        utxoInstance->DefineOwnProperty(
            *(utxoContext->context),
            String::NewFromUtf8(utxoContext->isolate, "value"),
            BigInt::New(utxoContext->isolate, utxos[i].value),
            DEFAULT_PROPERTY
        );
        
        utxoInstance->DefineOwnProperty(
            *(utxoContext->context),
            String::NewFromUtf8(utxoContext->isolate, "pubkeyhash"),
            String::NewFromUtf8(utxoContext->isolate, utxos[i].pubkeyhash),
            DEFAULT_PROPERTY
        );

        utxoInstance->DefineOwnProperty(
            *(utxoContext->context), 
            String::NewFromUtf8(utxoContext->isolate, "address"),
            String::NewFromUtf8(utxoContext->isolate, utxos[i].address),
            DEFAULT_PROPERTY
        );
        utxosInstance->Set(*(utxoContext->context), i, utxosInstance);
    } 

    (*(utxoContext->context))->Global()->DefineOwnProperty(
      *(utxoContext->context), String::NewFromUtf8(utxoContext->isolate, "_prevUtxos"),
      utxosInstance,
      DEFAULT_PROPERTY);
}

void NewPrevUtxoInstance(Isolate *isolate, Local<Context> context, void* address)
{
    if (utxoGet == NULL) {
        return;
    }

    UtxoContext utxoContext;
    utxoContext.isolate = isolate;
    utxoContext.context = &context;
    utxoGet(address, &utxoContext);
}
