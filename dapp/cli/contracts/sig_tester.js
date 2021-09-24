'use strict';

var Tester = function(){

};

Tester.prototype = {
    test:function(){
        SigVerification.verify();
    }
};
var tester = new Tester;