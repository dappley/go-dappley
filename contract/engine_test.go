package sc

import (
	"testing"
)



func TestScEngine_Execute(t *testing.T) {
	script:=
`'use strict';

var AddrChecker = function(){
	
};

AddrChecker.prototype = {
		check:function(addr,dummy){
    	return Blockchain.verifyAddress(addr)+dummy;
    }
};
var addrChecker = new AddrChecker;
`

	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.Execute("check","\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\",34")
}

func TestScEngine_Storage(t *testing.T) {
	script:=
		`'use strict';

var StorageTest = function(){
	
};

StorageTest.prototype = {
	set:function(key,value){
    	return LocalStorage.set(key,value);
    },
	get:function(key){
    	return LocalStorage.get(key);
    }
};
var storageTest = new StorageTest;
`
	ss := make(map[string]string)
	ss["key"]= "7"
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)
	sc.Execute("get","\"key\"")
}