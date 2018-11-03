package sc

import (
	"testing"
)



func TestScEngine_Execute(t *testing.T) {
	script:=
`'use strict';

var AddrChecker = function(){
	
};

AddrChecker.prototype = {
		check:function(addr){
    	return Blockchain.verifyAddress(addr);
    }
};
var addrChecker = new AddrChecker;
`

	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.Execute("check","\"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa\"")
}

func TestScEngine_Execute_FunctionCall(t *testing.T) {
	script:= `'use strict';
var Foo = function(){
	
};

Foo.prototype = {
		test:function(i,j){
    	return i-j;
    }
};

var f = new Foo();`

	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.Execute("test","1,9")
}

