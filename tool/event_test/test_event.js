'use strict';
var EventTest = function(){};
EventTest.prototype = {
    trigger: function(topic, data){
        _log.info("Received!topic:" + topic + ",data:" + data);
        return event.trigger(topic, data);
    },
    dapp_schedule: function(){
        _log.info("Running Dapp Scheduling");
    }
};
var eventTest = new EventTest;