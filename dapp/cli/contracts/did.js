'use strict';
var DIDManager=function(){};
DIDManager.prototype={ 
	new_DIDDocument:function(did, verificationMethods, authenticationMethods){ 
        var vmArray = []
        verificationMethods.forEach(vm => {
            var method = {
                id: vm[0],
                type: vm[1],
                controller: vm[2],
                key: vm[3],

            }
            vmArray.push(method)
        }
        )

        var amArray = []
        authenticationMethods.forEach(am => {
            var method = {
                id: am[0],
                type: am[1],
                controller: am[2],
                key: am[3],

            }
            amArray.push(method)
        }
        )
        var document = {
            id: did,
            verification_methods: vmArray,
            authentication_methods: amArray
        }
        LocalStorage.set(did, document)
    }
} 
module.exports=new DIDManager();