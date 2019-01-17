package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/stretchr/testify/assert"
)

var dummyAddr = "dummyAddr"

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
	script := `'use strict';
var MathTest = function(){};
MathTest.prototype = {
    transfer: function(to, amount, tip){
        return Blockchain.transfer(to, amount, tip);
    }
};
var transferTest = new MathTest;`

	contractPubKeyHash := core.NewContractPubKeyHash()
	contractAddr := contractPubKeyHash.GenerateAddress()
	contractUTXOs := []*core.UTXO{
		{
			Txid:     []byte("1"),
			TxIndex:  0,
			TXOutput: *core.NewTxOut(common.NewAmount(0), contractAddr, "somecontract"),
		},
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
			assert.Equal(t, []byte(contractPubKeyHash), sc.generatedTXs[0].Vin[0].PubKey)
		}
		if assert.Equal(t, 2, len(sc.generatedTXs[0].Vout)) {
			// payout
			assert.Equal(t, common.NewAmount(10), sc.generatedTXs[0].Vout[0].Value)
			// change
			assert.Equal(t, common.NewAmount(15-10-2), sc.generatedTXs[0].Vout[1].Value)

			assert.Equal(t, core.NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj"), sc.generatedTXs[0].Vout[0].PubKeyHash.GenerateAddress())
			assert.Equal(t, contractPubKeyHash, sc.generatedTXs[0].Vout[1].PubKeyHash)
		}
	}
}

func TestScEngine_StorageGet(t *testing.T) {
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

	ss := core.NewScState()
	ss.GetStorageByAddress(dummyAddr)["key"] = "7"
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportContractAddr(core.NewAddress(dummyAddr))
	sc.ImportLocalStorage(ss)
	assert.Equal(t, "7", sc.Execute("get", "\"key\""))
}

func TestScEngine_StorageSet(t *testing.T) {
	script := `'use strict';

var StorageTest = function(){
	
};

StorageTest.prototype = {
	set:function(key,value){
    	return LocalStorage.set(key, value);
    },
	get:function(key){
    	return LocalStorage.get(key);
    },
	setColor: function(key, color){
		var car = {type:"Fiat", model:"500", color:"white"};
		car.color = color;
		return LocalStorage.set(key, car);
	},
	getColor: function(key){
		return LocalStorage.get(key).color;
	}
};
var storageTest = new StorageTest;
`
	ss := core.NewScState()
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)
	sc.ImportContractAddr(core.NewAddress(dummyAddr))

	assert.Equal(t, "0", sc.Execute("set", "\"key\",6"))
	assert.Equal(t, "6", sc.Execute("get", "\"key\""))
	assert.Equal(t, "0", sc.Execute("set", "\"key\",\"abcd\""))
	assert.Equal(t, "abcd", sc.Execute("get", "\"key\""))
	assert.Equal(t, "0", sc.Execute("setColor", "\"key\",\"BLACK\""))
	assert.Equal(t, "BLACK", sc.Execute("getColor", "\"key\""))
}

func TestScEngine_StorageDel(t *testing.T) {
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
	ss := core.NewScState()
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)
	sc.ImportContractAddr(core.NewAddress(dummyAddr))
	assert.Equal(t, "0", sc.Execute("set", "\"key\",6"))
	assert.Equal(t, "0", sc.Execute("del", "\"key\""))
	assert.Equal(t, "null", sc.Execute("get", "\"key\""))
}

func TestScEngine_Reward(t *testing.T) {
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
	script :=
		`'use strict';

var TransactionTest = function(){
};

TransactionTest.prototype = {
	dump:function(dummy) {
		_log.error("dump")
		_log.error("tx id:", _tx.id)
		_log.error("prevUtxo length:", _prevUtxos.length)
		_log.error("tx vin length:", _tx.vin.length)
		let index = 0
		for (let vin of _tx.vin) {
				_log.error("Vin index:", index, " id:", vin.txid, " vout:", vin.vout, 
				    " signature:", vin.signature, " pubkey:", vin.pubkey)
				let prevUtxo = _prevUtxos[index]
				_log.error("PrevUtxo id:", prevUtxo.txid, " txIndex:", prevUtxo.txIndex, 
				    " value:", prevUtxo.value, " pubkeyhash:", prevUtxo.pubkeyhash, " address:", prevUtxo.address)
	    	}
		_log.error("tx vout length:", _tx.vin.length)
		index = 0
		for (let vout of _tx.vout) {
			_log.error("index:", index, " amount:", vout.amount, " pubkeyhash:", vout.pubkeyhash)
		}
	}
};
var transactionTest = new TransactionTest;
`
	ss := core.NewScState()
	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.ImportLocalStorage(ss)
	tx := core.MockTransaction()
	sc.ImportTransaction(tx)
	sc.ImportPrevUtxos(core.MockUtxos(tx.Vin))
	sc.Execute("dump", "\"dummy\"")
}

func TestStepRecord(t *testing.T) {
	script, _ := ioutil.ReadFile("jslib/step_recorder.js")

	reward := make(map[string]string)
	ss := core.NewScState()
	sc := NewV8Engine()
	sc.ImportSourceCode(string(script))
	sc.ImportLocalStorage(ss)
	sc.ImportContractAddr(core.NewAddress(dummyAddr))
	sc.ImportRewardStorage(reward)

	assert.Equal(t, "0", sc.Execute("record", "\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\", 20"))
	assert.Equal(t, "20", ss.GetStorageByAddress(core.NewAddress(dummyAddr).String())["dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
	assert.Equal(t, "20", reward["dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
	assert.Equal(t, "0", sc.Execute("record", "\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\", 15"))
	assert.Equal(t, "35", ss.GetStorageByAddress(core.NewAddress(dummyAddr).String())["dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
	assert.Equal(t, "35", reward["dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
	assert.Equal(t, "0", sc.Execute("record", "\"fastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\", 10"))
	assert.Equal(t, "10", ss.GetStorageByAddress(core.NewAddress(dummyAddr).String())["fastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
	assert.Equal(t, "10", reward["fastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
	assert.Equal(t, "35", ss.GetStorageByAddress(core.NewAddress(dummyAddr).String())["dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
	assert.Equal(t, "35", reward["dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"])
}

func TestCrypto_VerifySignature(t *testing.T) {
	script, _ := ioutil.ReadFile("test/test_crypto.js")

	sc := NewV8Engine()
	sc.ImportSourceCode(string(script))

	kp := core.NewKeyPair()
	msg := "hello world dappley"
	privData, _ := secp256k1.FromECDSAPrivateKey(&kp.PrivateKey)
	data := sha256.Sum256([]byte(msg))
	signature, _ := secp256k1.Sign(data[:], privData)

	assert.Equal(
		t,
		"true",
		sc.Execute("verifySig",
			fmt.Sprintf("\"%s\", \"%s\", \"%s\"",
				msg,
				hex.EncodeToString(kp.PublicKey),
				hex.EncodeToString(signature),
			),
		),
	)
}

func TestCrypto_VerifyPublicKey(t *testing.T) {
	script, _ := ioutil.ReadFile("test/test_crypto.js")

	sc := NewV8Engine()
	sc.ImportSourceCode(string(script))

	kp := core.NewKeyPair()
	fmt.Println(kp.PublicKey)
	pkh, err := core.NewUserPubKeyHash(kp.PublicKey)
	assert.Nil(t, err)
	addr := pkh.GenerateAddress()
	fmt.Println(addr)

	assert.Equal(
		t,
		"true",
		sc.Execute("verifyPk",
			fmt.Sprintf("\"%s\", \"%s\"",
				addr,
				hex.EncodeToString(kp.PublicKey),
			),
		),
	)
	assert.Equal(
		t,
		"false",
		sc.Execute("verifyPk",
			fmt.Sprintf("\"%s\", \"%s\"",
				"IncorrectAddress",
				hex.EncodeToString(kp.PublicKey),
			),
		),
	)
}

func TestMath(t *testing.T) {
	script, _ := ioutil.ReadFile("test/test_math.js")

	sc := NewV8Engine()
	sc.ImportSourceCode(string(script))
	sc.ImportSeed(10)

	res := sc.Execute("random", "20")
	assert.Equal(t, "14", res)
}

func TestBlkHeight(t *testing.T) {
	script, _ := ioutil.ReadFile("test/test_blockchain.js")

	sc := NewV8Engine()
	sc.ImportSourceCode(string(script))
	sc.ImportCurrBlockHeight(22334)

	assert.Equal(t, "22334", sc.Execute("getBlkHeight", ""))

}

func TestTrimWhiteSpaces(t *testing.T) {
	script, _ := ioutil.ReadFile("test/test_blockchain.js")
	scriptStr := string(script)
	str := strings.Replace(scriptStr, " ", "", -1)
	str = strings.Replace(str, "\\n", "", -1)
	fmt.Println(str)
}

func TestGetNodeAddress(t *testing.T) {
	script, _ := ioutil.ReadFile("test/test_blockchain.js")

	sc := NewV8Engine()
	sc.ImportSourceCode(string(script))
	sc.ImportNodeAddress(core.NewAddress("testAddr"))

	assert.Equal(t, "testAddr", sc.Execute("getNodeAddress", ""))
}

func TestNewAddress(t *testing.T) {
	kp := core.NewKeyPair()
	privData, _ := secp256k1.FromECDSAPrivateKey(&kp.PrivateKey)
	pk := hex.EncodeToString(privData)
	publicKey := hex.EncodeToString(kp.PublicKey)
	pkh, _ := core.NewUserPubKeyHash(kp.PublicKey)
	addr := pkh.GenerateAddress()
	fmt.Println("privatekey:", pk)
	fmt.Println("publickey:", publicKey)
	fmt.Println("addr:", addr)
}
