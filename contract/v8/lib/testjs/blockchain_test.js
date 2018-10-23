'use strict';

if (typeof _native_blockchain === "undefined") {
    throw new Error("_native_blockchain is undefined.");
}

var result = _native_blockchain.verifyAddress("70e30fcae5e7f4b2460faaa9e5b1bd912332ebb5");
console.log("verifyAddress:" + result)