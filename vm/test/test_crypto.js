'use strict';
var CryptoTest = function(){};
CryptoTest.prototype = {
    verifySig: function(msg, pubkey, sig){
        return crypto.verifySignature(msg, pubkey, sig);
    },
    verifyPk: function(addr, pubkey){
        return crypto.verifyPublicKey(addr, pubkey);
    }
};
module.exports = new CryptoTest();