'use strict';

var Blockchain = function () {
    Object.defineProperty(this, 'nativeBlockchain', {
        configurable: false,
        enumerable: false,
        get: function () {
            return _native_blockchain;
        }
    });
};
Blockchain.prototype = {
    verifyAddress: function (address) {
        return this.nativeBlockchain.verifyAddress(address);
    },
    transfer: function (to, amount, tip) {
        return this.nativeBlockchain.transfer(to, amount, tip);
    },
    getCurrBlockHeight: function () {
        return this.nativeBlockchain.getCurrBlockHeight();
    },
    deleteContract : function(){
        return this.nativeBlockchain.deleteContract();
    },
    dapp_schedule: function () {
    }
};
module.exports = new Blockchain();
