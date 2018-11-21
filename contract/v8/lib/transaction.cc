#include "transaction.h"
#include "../engine.h"

const PropertyAttribute DEFAULT_PROPERTY = static_cast<PropertyAttribute>(
    PropertyAttribute::DontDelete | PropertyAttribute::ReadOnly);

static FuncTransactionGet txGet = NULL;
typedef struct {
    Isolate *isolate;
    Local<Context> *context;
} TransactionContext;

void InitializeTransaction(FuncTransactionGet get) {
    txGet = get;
}

void SetTransactionData(struct transaction_t* tx, void* context) {
    if (tx == NULL || context == NULL) {
        return;
    }

    TransactionContext* txContext = static_cast<TransactionContext*>(context);

    Local<Object> txInstance = Object::New(txContext->isolate);
    txInstance->DefineOwnProperty(
        *(txContext->context),
        String::NewFromUtf8(txContext->isolate, "id"),
        String::NewFromUtf8(txContext->isolate, tx->id),
        DEFAULT_PROPERTY
    );

    Local<Array> vins = Array::New(txContext->isolate, tx->vin_length);
    for (int i = 0; i < tx->vin_length; i++) {
        Local<Object> vinInstance = Object::New(txContext->isolate);
        vinInstance->DefineOwnProperty(
            *(txContext->context),
            String::NewFromUtf8(txContext->isolate, "txid"),
            String::NewFromUtf8(txContext->isolate, tx->vin[i].txid),
            DEFAULT_PROPERTY
        );

        vinInstance->DefineOwnProperty(
            *(txContext->context),
            String::NewFromUtf8(txContext->isolate, "vout"),
            Integer::New(txContext->isolate, tx->vin[i].vout),
            DEFAULT_PROPERTY
        );
        
        vinInstance->DefineOwnProperty(
            *(txContext->context),
            String::NewFromUtf8(txContext->isolate, "signature"),
            String::NewFromUtf8(txContext->isolate, tx->vin[i].signature),
            DEFAULT_PROPERTY
        );

        vinInstance->DefineOwnProperty(
            *(txContext->context), 
            String::NewFromUtf8(txContext->isolate, "pubkey"),
            String::NewFromUtf8(txContext->isolate, tx->vin[i].pubkey),
            DEFAULT_PROPERTY
        );
        vins->Set(*(txContext->context), i, vinInstance);
    }
    txInstance->DefineOwnProperty(
        *(txContext->context),
        String::NewFromUtf8(txContext->isolate, "vin"),
        vins,
        DEFAULT_PROPERTY
    );

    Local<Array> vouts = Array::New(txContext->isolate, tx->vout_length);
    for (int i = 0; i < tx->vout_length; i++) {
        Local<Object> voutInstance = Object::New(txContext->isolate);
        voutInstance->DefineOwnProperty(
            *(txContext->context),
            String::NewFromUtf8(txContext->isolate, "amount"),
            BigInt::New(txContext->isolate, tx->vout[i].amount),
            DEFAULT_PROPERTY
        );

        voutInstance->DefineOwnProperty(
            *(txContext->context),
            String::NewFromUtf8(txContext->isolate, "pubkeyhash"),
            String::NewFromUtf8(txContext->isolate, tx->vout[i].pubkeyhash),
            DEFAULT_PROPERTY
        );
        vouts->Set(*(txContext->context), i, voutInstance);
    }
    txInstance->DefineOwnProperty(
        *(txContext->context),
        String::NewFromUtf8(txContext->isolate, "vout"),
        vouts,
        DEFAULT_PROPERTY
    );

    txInstance->DefineOwnProperty(
        *(txContext->context),
        String::NewFromUtf8(txContext->isolate, "tip"),
        BigInt::New(txContext->isolate, tx->tip),
        DEFAULT_PROPERTY
    );

    (*(txContext->context))->Global()->DefineOwnProperty(
      *(txContext->context), String::NewFromUtf8(txContext->isolate, "_tx"),
      txInstance,
      DEFAULT_PROPERTY);
}

void NewTransactionInstance(Isolate *isolate, Local<Context> context, void* address)
{
    if (txGet == NULL) {
        return;
    }

    TransactionContext txContext;
    txContext.isolate = isolate;
    txContext.context = &context;
    txGet(address, &txContext);
}
