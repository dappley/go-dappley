#ifndef _DAPPLEY_NF_VM_V8_LIB_FILE_H_
#define _DAPPLEY_NF_VM_V8_LIB_FILE_H_

#include <stddef.h>

#define MAX_PATH_LEN 1024
#define MAX_VERSION_LEN 64
#define MAX_VERSIONED_PATH_LEN 1088

/*#ifdef _WIN32
    #define FILE_SEPARATOR "\\"
    #define LIB_DIR     "\\lib"
    #define EXECUTION_FILE  "\\execution_env.js"
#else
    #define FILE_SEPARATOR "/"
    #define LIB_DIR     "/lib"
    #define EXECUTION_FILE  "/execution_env.js"
#endif //WIN32*/

char *readFile(const char *filepath, size_t *size);

bool isFile(const char *file);
bool getCurAbsolute(char *curCwd, int len);

#endif  // _DAPPLEY_NF_VM_V8_LIB_FILE_H_
