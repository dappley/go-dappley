package sc

import (
	"testing"
)

func TestScEngine_Execute(t *testing.T) {
	sc := NewScEngine(`'use strict';

if (typeof _native_blockchain === "undefined") {
    throw new Error("_native_blockchain is undefined.");
}

var result = _native_blockchain.verifyAddress("1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV");
"verifyAddress:" + result`)
	sc.Execute()
}