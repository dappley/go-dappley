package sc

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t,"35",sc.Execute("check","\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\",34"))
}

func TestScEngine_StorageGet(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
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
	assert.Equal(t,"7", sc.Execute("get","\"key\""))
}

func TestScEngine_StorageSet(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
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
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)

	assert.Equal(t,"0", sc.Execute("set","\"key\",6"))
	assert.Equal(t,"6", sc.Execute("get","\"key\""))
}

func TestScEngine_StorageDel(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
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
    },
	del:function(key){
    	return LocalStorage.del(key);
    }
};
var storageTest = new StorageTest;
`
	ss := make(map[string]string)
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)
	assert.Equal(t,"0", sc.Execute("set","\"key\",6"))
	assert.Equal(t,"0", sc.Execute("del","\"key\""))
	assert.Equal(t,"null", sc.Execute("get","\"key\""))
}