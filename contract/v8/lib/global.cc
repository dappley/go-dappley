#include "global.h"
#include "blockchain.h"
#include "event.h"
#include "instruction_counter.h"
#include "storage.h"
#include "event.h"
#include "logger.h"
#include "transaction.h"
#include "reward_distributor.h"
#include "prev_utxo.h"
#include "crypto.h"
#include "math.h"
#include "require_callback.h"

Local<ObjectTemplate> CreateGlobalObjectTemplate(Isolate *isolate) {
  Local<ObjectTemplate> globalTpl = ObjectTemplate::New(isolate);
  globalTpl->SetInternalFieldCount(1);

  NewNativeRequireFunction(isolate, globalTpl);
//  NewNativeLogFunction(isolate, globalTpl);
//  NewNativeEventFunction(isolate, globalTpl);
  // NewNativeRandomFunction(isolate, globalTpl);

//  NewStorageType(isolate, globalTpl);

  return globalTpl;
}

void SetGlobalObjectProperties(Isolate *isolate, Local<Context> context,
                               V8Engine *e, void *handler) {
  // set e to global.
  Local<Object> global = context->Global();
  global->SetInternalField(0, External::New(isolate, e));

    NewBlockchainInstance(isolate, context, (void *)handler);
    NewCryptoInstance(isolate, context, (void *)handler);
    NewStorageInstance(isolate, context, (void *)handler);
    NewLoggerInstance(isolate, context, (void *)handler);
    NewTransactionInstance(isolate, context, (void *)handler);
    NewRewardDistributorInstance(isolate, context, (void *)handler);
    NewPrevUtxoInstance(isolate, context, (void *)handler);
    NewMathInstance(isolate, context, (void *)handler);
    NewEventInstance(isolate, context, (void *)handler);

    NewInstructionCounterInstance(isolate, context,
                                    &(e->stats.count_of_executed_instructions), e);
//  uint64_t build_flag = e->ver;
//  if (BUILD_MATH == (build_flag & BUILD_MATH)) {
//    NewRandomInstance(isolate, context, lcsHandler);
//  }
//  if (BUILD_BLOCKCHAIN == (build_flag & BUILD_BLOCKCHAIN)) {
//    NewBlockchainInstance(isolate, context, lcsHandler, build_flag);
//  }
  
//  NewCryptoInstance(isolate, context);
}

V8Engine *GetV8EngineInstance(Local<Context> context) {
  Local<Object> global = context->Global();
  Local<Value> val = global->GetInternalField(0);

  if (!val->IsExternal()) {
    return NULL;
  }

  return static_cast<V8Engine *>(Local<External>::Cast(val)->Value());
}
