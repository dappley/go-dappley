'use strict';

var SigVerification = function () {
    Object.defineProperty(this, 'verifier', {
        configurable: false,
        enumerable: false,
        get: function () {
            return _verifier;
        }
    });
};

SigVerification.prototype = {
    verify: function(){
        LocalStorage.set('test', 'yay')
    }
}

var SigVerification = new SigVerification();