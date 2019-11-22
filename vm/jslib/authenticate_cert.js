'use strict';

var LocalAuthenCert = function () {
    Object.defineProperty(this, 'authenCert', {
        configurable: false,
        enumerable: false,
        get: function () {
            return _native_authenticate_cert;
        }
    });
};

LocalAuthenCert.prototype = {
    authenInit: function (cert) {
        return this.authenCert.authenInit(cert);
    },

    hashStorage: function (key , data) {
        var resultVerify = this.authenCert.authenVerify();
        if (resultVerify == true){
            LocalStorage.set(key,data);
            return true;
        }
        return false;
    },

    hashCheck: function (key) {
        var result = LocalStorage.get(key)
        return result
    },

    dapp_schedule: function () {
        
    }
};

module.exports = new LocalAuthenCert();