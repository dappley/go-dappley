#include "transaction.h"
#include "../engine.h"

const PropertyAttribute DEFAULT_PROPERTY = static_cast<PropertyAttribute>(
    PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly);

static FuncTransactionGet txGet = NULL;

void InitializeTransaction(FuncTransactionGet get) {
    txGet = get;
}

void NewTransactionInstance(Isolate *isolate, Local<Context> context, void* address)
{
    if (txGet == NULL) {
        return;
    }

    transaction_t*  tx = txGet(address);
    if (tx == NULL) {
        return;
    }

    Local<Object> txInstance = Object::New(isolate);
    txInstance->DefineOwnProperty(
        context,
        String::NewFromUtf8(isolate, "id"),
        String::NewFromUtf8(isolate, tx->id),
        DEFAULT_PROPERTY
    );
    free(tx->id);

    Local<Array> vins = Array::New(isolate, tx->vin_length);
    for (int i = 0; i < tx->vin_length; i++) {
        Local<Object> vinInstance = Object::New(isolate);
        vinInstance->DefineOwnProperty(
            context,
            String::NewFromUtf8(isolate, "txid"),
            String::NewFromUtf8(isolate, tx->vin[i].txid),
            DEFAULT_PROPERTY
        );
        free(tx->vin[i].txid);

        vinInstance->DefineOwnProperty(
            context,
            String::NewFromUtf8(isolate, "vout"),
            Integer::New(isolate, tx->vin[i].vout),
            DEFAULT_PROPERTY
        );
        
        vinInstance->DefineOwnProperty(
            context,
            String::NewFromUtf8(isolate, "signature"),
            String::NewFromUtf8(isolate, tx->vin[i].signature),
            DEFAULT_PROPERTY
        );
        free(tx->vin[i].signature);

        vinInstance->DefineOwnProperty(
            context, 
            String::NewFromUtf8(isolate, "pubkey"),
            String::NewFromUtf8(isolate, tx->vin[i].pubkey),
            DEFAULT_PROPERTY
        );
        vins->Set(context, i, vinInstance);
        free(tx->vin[i].pubkey);
    }
    txInstance->DefineOwnProperty(
        context,
        String::NewFromUtf8(isolate, "vin"),
        vins,
        DEFAULT_PROPERTY
    );

    Local<Array> vouts = Array::New(isolate, tx->vout_length);
    for (int i = 0; i < tx->vout_length; i++) {
        Local<Object> voutInstance = Object::New(isolate);
        voutInstance->DefineOwnProperty(
            context,
            String::NewFromUtf8(isolate, "amount"),
            BigInt::New(isolate, tx->vout[i].amount),
            DEFAULT_PROPERTY
        );

        voutInstance->DefineOwnProperty(
            context,
            String::NewFromUtf8(isolate, "pubkeyhash"),
            String::NewFromUtf8(isolate, tx->vout[i].pubkeyhash),
            DEFAULT_PROPERTY
        );
        free(tx->vout[i].pubkeyhash);
        vouts->Set(context, i, voutInstance);
    }
    txInstance->DefineOwnProperty(
        context,
        String::NewFromUtf8(isolate, "vout"),
        vouts,
        DEFAULT_PROPERTY
    );

    txInstance->DefineOwnProperty(
        context,
        String::NewFromUtf8(isolate, "tip"),
        BigInt::New(isolate, tx->tip),
        DEFAULT_PROPERTY
    );

    context->Global()->DefineOwnProperty(
      context, String::NewFromUtf8(isolate, "_tx"),
      txInstance,
      DEFAULT_PROPERTY);
}
