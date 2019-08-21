#include "file.h"
#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include "memory.h"

char *readFile(const char *filepath, size_t *size) {
    if (size != NULL) {
        *size = 0;
    }

    FILE *f = fopen(filepath, "r");
    if (f == NULL) {
        return NULL;
    }

    // get file size.
    fseek(f, 0L, SEEK_END);
    size_t file_size = ftell(f);
    rewind(f);

    char *data = (char *)MyMalloc(file_size + 1);
    size_t idx = 0;

    size_t len = 0;
    while ((len = fread(data + idx, sizeof(char), file_size + 1 - idx, f)) > 0) {
        idx += len;
    }
    *(data + idx) = '\0';

    if (feof(f) == 0) {
        MyFree(static_cast<void *>(data));
        return NULL;
    }

    fclose(f);

    if (size != NULL) {
        *size = file_size;
    }

    return data;
}

bool isFile(const char *file) {
    struct stat buf;
    if (stat(file, &buf) != 0) {
        return false;
    }
    if (S_ISREG(buf.st_mode)) {
        return true;
    } else {
        return false;
    }
}

bool getCurAbsolute(char *curCwd, int len) {
    char tmp[MAX_VERSIONED_PATH_LEN] = {0};
    if (!getcwd(tmp, MAX_VERSIONED_PATH_LEN)) {
        return false;
    }

    strncat(tmp, "/jslib/execution_env.js", MAX_VERSIONED_PATH_LEN - strlen(tmp) - 1);

    char *pc = realpath(tmp, NULL);
    if (pc == NULL) {
        return false;
    }
    int pcLen = strlen(pc);
    if (pcLen >= len) {
        free(pc);
        return false;
    }
    memcpy(curCwd, pc, pcLen - strlen("/execution_env.js"));
    // strncpy(curCwd, pc, len - 1);
    curCwd[pcLen - strlen("/execution_env.js")] = 0x00;
    free(pc);
    return true;
}
