'use strict';
var EventTest = function(){};
EventTest.prototype = {
    trigger: function(topic, data){
        _log.info("Received!topic:" + topic + ",data:" + data);
        return event.trigger(topic, data);
    }
};
var eventTest = new EventTest;