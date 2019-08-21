'use strict';
var EventTest = function(){};
EventTest.prototype = {
    trigger: function(topic, data){
        return event.trigger(topic, data);
    }
};
module.exports = new EventTest();