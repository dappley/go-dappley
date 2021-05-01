'usestrict';

var StepRecorder=function(){};
StepRecorder.prototype={
    record:function(addr,steps){
        var originalSteps=LocalStorage.get(addr);
        LocalStorage.set(addr,originalSteps+steps);
        return _native_reward.record(addr,steps);}
};

module.exports = new StepRecorder();