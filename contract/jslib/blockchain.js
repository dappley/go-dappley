'use strict';

var Blockchain = function () {
    Object.defineProperty(this, "nativeBlockchain", {
        configurable: false,
        enumerable: false,
        get: function(){
            return _native_blockchain;
        }
    });
};

Blockchain.prototype = {
    verifyAddress: function (address) {
        return this.nativeBlockchain.verifyAddress(address);
    }
};

Blockchain.prototype = {
    transfer: function (to, amount, tip) {
        return this.nativeBlockchain.transfer(to, amount, tip);
    }
};

var Blockchain = new Blockchain();
