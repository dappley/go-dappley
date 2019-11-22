'use strict';

var HashStorageTest = function () {
};

HashStorageTest.prototype = {
    authenInit: function (cert) {
        var resultInit = LocalAuthenCert.authenInit(cert);
        return resultInit;
    },
    
    hashStorage: function (key , data) {
        var resultVerify = LocalAuthenCert.authenVerify();
        if (resultVerify == true){
            LocalStorage.set(key,data)
            return true
        }
        return false;
    },
    
    hashCheck: function (key) {
        var result = LocalStorage.get(key)
        return result
    }
};

module.exports = new HashStorageTest();