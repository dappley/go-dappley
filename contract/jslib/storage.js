'use strict';

var LocalStorage = function () {
    Object.defineProperty(this, "nativeStorage", {
        configurable: false,
        enumerable: false,
        get: function () {
            return _native_storage;
        }
    });
};

LocalStorage.prototype = {
    get: function(key){
        var value = this.nativeStorage.get(key);
        if (value != null){
            value = JSON.parse(value);
        }
        return value;
    },

    set: function(key, value){
        return this.nativeStorage.set(key, JSON.stringify(value));
    },

    del: function(key) {
        var value = this.nativeStorage.del(key);
        if (value != 0) {
            throw new Error ("Delete failed. key: " + key);
        }
        return value;
    }
};

var LocalStorage = new LocalStorage();