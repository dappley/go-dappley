'use strict';

var Add = function () {
};

Add.prototype = {
    add: function (a, b) {
        return a + b;
    },
    dapp_schedule: function () {
    }
};

module.exports = new Add();