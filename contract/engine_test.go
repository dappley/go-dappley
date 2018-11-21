package sc

import (
	"testing"

	"github.com/dappley/go-dappley/core"
	"github.com/sirupsen/logrus"
)

func TestScEngine_Execute(t *testing.T) {
	script :=
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
	sc.Execute("check", "\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\",34")
}

func TestScEngine_StorageGet(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script :=
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
	ss["key"] = "7"
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)
	sc.Execute("get", "\"key\"")
}

func TestScEngine_StorageSet(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script :=
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
	sc.Execute("set", "\"key\",6")
	sc.Execute("get", "\"key\"")
}

func TestScEngine_StorageDel(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script :=
		`'use strict';

var StorageTest = function(){
	
};

StorageTest.prototype = {
	set:function(key,value){
		_log.error("Test case in Storage del ", "set")
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
	sc.Execute("set", "\"key\",6")
	sc.Execute("del", "\"key\"")
	sc.Execute("get", "\"key\"")
}

func TestScEngine_TransactionTest(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script :=
		`'use strict';

var TransactionTest = function(){
};

TransactionTest.prototype = {
	dump:function(dummy) {
		_log.error("dump")
		_log.error("tx id:", _tx.id)
		_log.error("tx vin length:", _tx.vin.length)
		let index = 0
		for (let vin of _tx.vin) {
			_log.error("index:", index, " id: ", vin.txid, " vout: ", vin.vout, " signature: ", vin.signature, " pubkey: ", vin.pubkey)
		}
		_log.error("tx vout length:", _tx.vin.length)
		index = 0
		for (let vout of _tx.vout) {
			_log.error("index:", index, " amount: ", vout.amount, " pubkeyhash: ", vout.pubkeyhash)
		}
	}
};
var transactionTest = new TransactionTest;
`
	ss := make(map[string]string)
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)
	sc.ImportTransaction(core.MockTransaction())
	sc.Execute("dump", "\"dummy\"")
}
