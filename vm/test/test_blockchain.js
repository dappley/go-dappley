'use strict';
var BlkHeightTest = function(){};
BlkHeightTest.prototype = {
    getBlkHeight: function(){
        return Blockchain.getCurrBlockHeight();
    },
};
module.exports = new BlkHeightTest();