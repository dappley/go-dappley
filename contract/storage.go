package sc
import "C"

var ScStorageTmpDb = make(map[string]string)

//export StorageGetFunc
func StorageGetFunc(key *C.char) *C.char{
	return C.CString(ScStorageTmpDb[C.GoString(key)])
}

//export StorageSetFunc
func StorageSetFunc(key,value *C.char) int{
	ScStorageTmpDb[C.GoString(key)] = C.GoString(value)
	return 0
}

//export StorageDelFunc
func StorageDelFunc(key *C.char) int{
	return 0
}