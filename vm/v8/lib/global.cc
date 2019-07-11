#include "global.h"
#include "blockchain.h"
#include "crypto.h"
#include "event.h"
#include "instruction_counter.h"
#include "logger.h"
#include "math.h"
#include "prev_utxo.h"
#include "require_callback.h"
#include "reward_distributor.h"
#include "storage.h"
#include "transaction.h"

Local<ObjectTemplate> CreateGlobalObjectTemplate(Isolate *isolate) {
    Local<ObjectTemplate> globalTpl = ObjectTemplate::New(isolate);
    globalTpl->SetInternalFieldCount(1);

    NewNativeRequireFunction(isolate, globalTpl);

    return globalTpl;
}

void SetGlobalObjectProperties(Isolate *isolate, Local<Context> context, V8Engine *e, void *handler) {
    // set e to global.
    Local<Object> global = context->Global();
    global->SetInternalField(0, External::New(isolate, handler));

    NewBlockchainInstance(isolate, context, (void *)handler);
    NewCryptoInstance(isolate, context, (void *)handler);
    NewStorageInstance(isolate, context, (void *)handler);
    NewLoggerInstance(isolate, context, (void *)handler);
    NewTransactionInstance(isolate, context, (void *)handler);
    NewRewardDistributorInstance(isolate, context, (void *)handler);
    NewPrevUtxoInstance(isolate, context, (void *)handler);
    NewMathInstance(isolate, context, (void *)handler);
    NewEventInstance(isolate, context, (void *)handler);

    NewInstructionCounterInstance(isolate, context, &(e->stats.count_of_executed_instructions), e);
}

void *GetV8EngineHandler(Local<Context> context) {
    Local<Object> global = context->Global();
    Local<Value> val = global->GetInternalField(0);

    if (!val->IsExternal()) {
        return NULL;
    }
    return static_cast<void *>(Local<External>::Cast(val)->Value());
}
