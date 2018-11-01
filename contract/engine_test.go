package sc

import (
	"testing"
)

func TestScEngine_Execute(t *testing.T) {
	sc := NewV8Engine()
	sc.ImportSourceCode(`'use strict';

if (typeof _native_blockchain === "undefined") {
    throw new Error("_native_blockchain is undefined.");
}

var result = _native_blockchain.verifyAddress("1G4r54VdJsotfCukXUWmg1ZRnhj2s6TvbV");
"verifyAddress:" + result`)
	sc.Execute()
}