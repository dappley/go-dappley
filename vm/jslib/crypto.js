'use strict';
var Crypto = function(){};
Crypto.prototype = {
    verifySig: function(msg, pubkey, sig){
        return crypto.verifySignature(msg, pubkey, sig);
    },
    verifyPk: function(addr, pubkey){
        return crypto.verifyPublicKey(addr, pubkey);
    }
};
var Crypto = new Crypto();