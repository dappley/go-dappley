'use strict';
var MathTest = function(){};
MathTest.prototype = {
    random: function(max){
        return math.random(max);
    }
};
var mathTest = new MathTest;