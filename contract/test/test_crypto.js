'use strict';
var CryptoTest = function(){};
CryptoTest.prototype = {
    verifySig: function(msg, pubkey, sig){
        return crypto.verifySignature(msg, pubkey, sig);
    }
};
var cryptoTest = new CryptoTest;