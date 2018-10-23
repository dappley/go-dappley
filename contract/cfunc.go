package contract


/*
#include <stdbool.h>

bool VerifyAddressFunc(const char* address);
bool VerifyAddressFunc_cgo(const char *address) {
	return VerifyAddressFunc(address);
};
*/
import "C"
