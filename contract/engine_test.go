package sc

import (
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestScEngine_Execute(t *testing.T) {
	script := `'use strict';

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
	assert.Equal(t, "35", sc.Execute("check", "\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\",34"))
}

func TestScEngine_Execute_SyntaxError(t *testing.T) {
	// Missing quotes around 'use strict'
	script := `use strict;

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
	assert.Equal(t, "1", sc.Execute("check", "\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\",34"))
}

func TestScEngine_BlockchainTransfer(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script := `'use strict';
var TransferTest = function(){};
TransferTest.prototype = {
    transfer: function(to, amount, tip){
        return Blockchain.transfer(to, amount, tip);
    }
};
var transferTest = new TransferTest;`

	contractPubKeyHash := core.NewContractPubKeyHash()
	contractAddr := contractPubKeyHash.GenerateAddress()
	contractUTXOs := []*core.UTXO{
		{
			Txid:     []byte("1"),
			TxIndex:  1,
			TXOutput: *core.NewTxOut(common.NewAmount(15), contractAddr, ""),
		},
		{
			Txid:     []byte("2"),
			TxIndex:  0,
			TXOutput: *core.NewTxOut(common.NewAmount(3), contractAddr, ""),
		},
	}

	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportContractAddr(contractAddr)
	sc.ImportSourceTXID([]byte("thatTX"))
	sc.ImportUTXOs(contractUTXOs)
	result := sc.Execute("transfer", "'16PencPNnF8CiSx2EBGEd1axhf7vuHCouj','10','2'")

	assert.Equal(t, "0", result)
	if assert.Equal(t, 1, len(sc.generatedTXs)) {
		if assert.Equal(t, 1, len(sc.generatedTXs[0].Vin)) {
			assert.Equal(t, []byte("1"), sc.generatedTXs[0].Vin[0].Txid)
			assert.Equal(t, 1, sc.generatedTXs[0].Vin[0].Vout)
			assert.Equal(t, []byte("thatTX"), sc.generatedTXs[0].Vin[0].Signature)
			assert.Equal(t, contractPubKeyHash.GetPubKeyHash(), sc.generatedTXs[0].Vin[0].PubKey)
		}
		if assert.Equal(t, 2, len(sc.generatedTXs[0].Vout)) {
			// payout
			assert.Equal(t, common.NewAmount(10), sc.generatedTXs[0].Vout[0].Value)
			// change
			assert.Equal(t, common.NewAmount(15-10-2), sc.generatedTXs[0].Vout[1].Value)

			assert.Equal(t, core.NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj"), sc.generatedTXs[0].Vout[0].PubKeyHash.GenerateAddress())
			assert.Equal(t, contractPubKeyHash.GetPubKeyHash(), sc.generatedTXs[0].Vout[1].PubKeyHash.GetPubKeyHash())
		}
	}
}

func TestScEngine_StorageGet(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script := `'use strict';

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
	assert.Equal(t, "7", sc.Execute("get", "\"key\""))
}

func TestScEngine_StorageSet(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script := `'use strict';

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

	assert.Equal(t, "0", sc.Execute("set", "\"key\",6"))
	assert.Equal(t, "6", sc.Execute("get", "\"key\""))
}

func TestScEngine_StorageDel(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script := `'use strict';

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
	assert.Equal(t, "0", sc.Execute("set", "\"key\",6"))
	assert.Equal(t, "0", sc.Execute("del", "\"key\""))
	assert.Equal(t, "null", sc.Execute("get", "\"key\""))
}

func TestScEngine_Reward(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	script :=
		`'use strict';

var RewardTest = function(){
	
};

RewardTest.prototype = {
	reward:function(addr,amount){
    	return _native_reward.record(addr,amount);
    }
};
var rewardTest = new RewardTest;
`
	ss := make(map[string]string)
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportRewardStorage(ss)

	assert.Equal(t, "0", sc.Execute("reward", "\"myAddr\",\"8\""))
	assert.Equal(t, "0", sc.Execute("reward", "\"myAddr\",\"9\""))
	assert.Equal(t, "17", ss["myAddr"])
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
