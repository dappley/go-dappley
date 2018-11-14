'use strict';

var StepRecorder = function(){

};

StepRecorder.prototype = {
    record:function(addr,steps){
        return _native_reward.record(addr,steps);
    }
};
var stepRecorder = new StepRecorder;