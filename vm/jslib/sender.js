'use strict';

var Sender = function () {
    Object.defineProperty(this, "sender", {
        configurable: false,
        enumerable: false,
        get: function(){
            return _tx;
        }
    });
};

Sender.prototype = {
    getAddress : function(){
        return this.sender.vin[0].pubkey;
    }
};

var Sender = new Sender();
