#ifndef _DAPPLEY_NF_VM_V8_ALLOCATOR_H_
#define _DAPPLEY_NF_VM_V8_ALLOCATOR_H_

#include <stdint.h>
#include <v8.h>

using namespace v8;

class ArrayBufferAllocator : public ArrayBuffer::Allocator {
public:
  ArrayBufferAllocator();
  virtual ~ArrayBufferAllocator();

  /**
   * Allocate |length| bytes. Return NULL if allocation is not successful.
   * Memory should be initialized to zeroes.
   */
  virtual void *Allocate(size_t length);

  /**
   * Allocate |length| bytes. Return NULL if allocation is not successful.
   * Memory does not have to be initialized.
   */
  virtual void *AllocateUninitialized(size_t length);

  /**
   * Free the memory block of size |length|, pointed to by |data|.
   * That memory is guaranteed to be previously allocated by |Allocate|.
   */
  virtual void Free(void *data, size_t length);

  size_t total_available_size();

  size_t peak_allocated_size();

private:
  size_t total_allocated_size_;
  size_t peak_allocated_size_;
};

#endif
