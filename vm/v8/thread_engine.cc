#include "engine.h"
#include "engine_int.h"
#include "lib/logger.h"
#include "lib/tracing.h"
#include "lib/vm_error.h"

#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/time.h>
#include <unistd.h>
#include <thread>
#define MicroSecondDiff(newtv, oldtv) (1000000 * (unsigned long long)((newtv).tv_sec - (oldtv).tv_sec) + (newtv).tv_usec - (oldtv).tv_usec)  //微秒
#define CodeExecuteErr 1
#define CodeExecuteInnerVmErr 2
#define CodeTimeOut 3

void SetRunScriptArgs(v8ThreadContext *ctx, V8Engine *e, int opt, const char *source, int line_offset, int allow_usage) {
    ctx->e = e;
    ctx->input.source = source;
    ctx->input.opt = (OptType)opt;
    ctx->input.allow_usage = allow_usage;
    ctx->input.line_offset = line_offset;
}

char *RunInjectTracingInstructionsThread(V8Engine *e, const char *source, int *source_line_offset, int allow_usage, uintptr_t handler) {
    v8ThreadContext ctx;
    memset(&ctx, 0x00, sizeof(ctx));
    SetRunScriptArgs(&ctx, e, INSTRUCTION, source, *source_line_offset, allow_usage);
    ctx.input.handler = handler;
    bool btn = CreateScriptThread(&ctx);
    if (btn == false) {
        printf("Failed to create script thread");
        return NULL;
    }
    *source_line_offset = ctx.output.line_offset;
    return ctx.output.result;
}

int RunV8ScriptThread(char **result, V8Engine *e, const char *source, int source_line_offset, uintptr_t handler) {
    v8ThreadContext ctx;
    memset(&ctx, 0x00, sizeof(ctx));
    SetRunScriptArgs(&ctx, e, RUNSCRIPT, source, source_line_offset, 1);
    ctx.input.handler = handler;

    bool btn = CreateScriptThread(&ctx);
    if (btn == false) {
        return VM_UNEXPECTED_ERR;
    }

    *result = ctx.output.result;
    return ctx.output.ret;
}

void *ThreadExecutor(void *args) {
    v8ThreadContext *ctx = (v8ThreadContext *)args;
    if (ctx->input.opt == INSTRUCTION) {
        TracingContext tContext;
        tContext.source_line_offset = 0;
        tContext.tracable_source = NULL;
        tContext.strictDisallowUsage = ctx->input.allow_usage;
        ExecuteByV8(ctx->input.source, 0, ctx->input.handler, NULL, ctx->e, InjectTracingInstructionDelegate, (void *)&tContext);

        ctx->output.line_offset = tContext.source_line_offset;
        ctx->output.result = static_cast<char *>(tContext.tracable_source);
    } else {
        ctx->output.ret =
            ExecuteByV8(ctx->input.source, ctx->input.line_offset, ctx->input.handler, &ctx->output.result, ctx->e, ExecuteSourceDataDelegate, NULL);
    }

    ctx->is_finished = true;
    return 0x00;
}
// return : success return true. if hava err ,then return false. and not need to free heap
// if gettimeofday hava err ,There is a risk of an infinite loop
bool CreateScriptThread(v8ThreadContext *ctx) {
    pthread_t thread;
    pthread_attr_t attribute;
    pthread_attr_init(&attribute);
    pthread_attr_setstacksize(&attribute, 2 * 1024 * 1024);
    pthread_attr_setdetachstate(&attribute, PTHREAD_CREATE_DETACHED);
    struct timeval tcBegin, tcEnd;
    int rtn = gettimeofday(&tcBegin, NULL);
    if (rtn != 0) {
        printf("CreateScriptThread get start time err:%d\n", rtn);
        return false;
    }
    rtn = pthread_create(&thread, &attribute, ThreadExecutor, (void *)ctx);
    if (rtn != 0) {
        printf("CreateScriptThread pthread_create err:%d\n", rtn);
        return false;
    }

    int timeout = ctx->e->timeout;
    bool is_kill = false;
    // thread safe
    while (1) {
        if (ctx->is_finished == true) {
            if (is_kill == true) {
                ctx->output.ret = VM_EXE_TIMEOUT_ERR;
            } else if (ctx->e->is_inner_vm_error_happen == 1) {
                ctx->output.ret = VM_INNER_EXE_ERR;
            }
            break;
        } else {
            usleep(10);  // 10 micro second loop .epoll_wait optimize
            rtn = gettimeofday(&tcEnd, NULL);
            if (rtn) {
                printf("CreateScriptThread get end time err:%d\n", rtn);
                continue;
            }
            int diff = MicroSecondDiff(tcEnd, tcBegin);

            if (diff >= timeout && is_kill == false) {
                printf("CreateScriptThread timeout timeout:%d diff:%d\n", timeout, diff);
                TerminateExecution(ctx->e);
                is_kill = true;
            }
        }
    }
    return true;
}