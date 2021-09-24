'use strict';

var Tester = function(){

};

Tester.prototype = {
    test:function(){
        LocalStorage.set("test1", "yay1");
        SigVerification.verify();
    }
};
module.exports = new Tester;