'use strict';

var Add = function () {
};

Add.prototype = {
    add: function (a, b) {
        return a + b;
    }
};

module.exports = new Add();