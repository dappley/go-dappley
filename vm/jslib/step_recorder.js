'use strict';

var StepRecorder = function () {
};

StepRecorder.prototype = {
    record: function (key, value) {
        var originalSteps = LocalStorage.get(key);
        if (originalSteps === null){
            originalSteps = 0;
        }
        LocalStorage.set(key, parseInt(originalSteps) + parseInt(value));
        return _native_reward.record(key, value);
    },
    delete: function (key) {
        LocalStorage.del(key);
    }
};
module.exports = new StepRecorder();

