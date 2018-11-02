package sc

import (
	"testing"
)



func TestScEngine_Execute(t *testing.T) {
	script:= `'use strict';

if (typeof Blockchain === "undefined") {
throw new Error("_native_blockchain is undefined.");
}

var result = Blockchain.verifyAddress("1G4r54VdJsotfCukXUWmg1ZRnhj2s6TvbV");
"verifyAddress:" + result`

	sc := NewV8Engine()
	sc.ImportSourceCode(script)
	sc.Execute()
}
