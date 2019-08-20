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
var blkHeightTest = new BlkHeightTest;