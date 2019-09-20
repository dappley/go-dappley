'use strict';
var BlkHeightTest = function(){};
BlkHeightTest.prototype = {
    getBlkHeight: function(){
        return Blockchain.getCurrBlockHeight();
    },
    getNodeAddress: function(){
        return Blockchain.getNodeAddress();
    }
};
module.exports = new BlkHeightTest();