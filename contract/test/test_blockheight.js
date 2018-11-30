'use strict';
var BlkHeightTest = function(){};
BlkHeightTest.prototype = {
    getBlkHeight: function(){
        return Blockchain.getCurrBlockHeight();
    },
};
var blkHeightTest = new BlkHeightTest;