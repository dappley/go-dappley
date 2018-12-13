#ifndef __MEMORY_H__
#define __MEMORY_H__

#include <stddef.h>

void* MyMalloc(size_t size);
void  MyFree(void* data);

#endif /* __MEMORY_H__ */
