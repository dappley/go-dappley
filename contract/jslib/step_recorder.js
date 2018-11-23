'use strict';

var StepRecorder = function(){

};

StepRecorder.prototype = {
    record: function(addr, steps){
        var originalSteps = LocalStorage.get(addr);
        LocalStorage.set(addr, originalSteps + steps)
        _native_reward.record(addr, steps);
    }
};

var stepRecorder = new StepRecorder;