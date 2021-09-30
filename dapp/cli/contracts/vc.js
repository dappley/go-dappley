'use strict';

function MatchesSchema (inputObject, schema){
    const inputKeys = Object.keys(inputObject);
    const schemaKeys = Object.keys(schema);
    const allKeys = inputKeys.concat(schemaKeys);
    const union = new Set(allKeys);
    if (union.size !== inputKeys.length || union.size !== schemaKeys.length){
        return false;
    }
    for (const property in inputObject) {
        var iPropObject = false;
        var sPropObject = false;
        iPropObject = (typeof inputObject[property] === 'object' && !Array.isArray(inputObject[property]) && inputObject[property] !== null);
        sPropObject = (typeof schema[property] === 'object' && !Array.isArray(schema[property]) && schema[property] !== null);
        if (iPropObject !== sPropObject){
            return false;
        }
        if (iPropObject){
            if (!MatchesSchema(inputObject[property], schema[property])){
                return false;
            }
        }
    }
    return true;
}

var VCStorage = function(){

};

VCStorage.prototype = {
    addDID:function(doc){
        if (!("id" in doc)){
            throw 'id missing';
        }

        if (LocalStorage.get(doc.id) !== null){
            throw 'did already exists';
        }

        if ("verificationMethod" in doc){
            if (!Array.isArray(doc.verificationMethod)){
                throw 'verificationMethod is not an array';
            }
            for (var i = 0; i < doc.verificationMethod.length; i++){
                if (!("id" in doc.verificationMethod[i] && "controller" in doc.verificationMethod[i] && "type" in doc.verificationMethod[i])){
                    throw 'verificationMethod formatted incorrectly';
                }
            }
        }
        LocalStorage.set(doc.id, doc);
    },
    updateDID:function(did, doc, msg, sig){
        var currentDoc = LocalStorage.get(did);
        if (currentDoc === null){
            throw 'did does not exist';
        }
        if (!("verificationMethod" in currentDoc)){
            throw 'document has no verification methods'
        }
        for (var i = 0; i < currentDoc.verificationMethod.length; i++){
            if ("publicKeyHex" in currentDoc.verificationMethod[i]){
            var pubKey = currentDoc.verificationMethod[i].publicKeyHex;
            if (Crypto.verifySig(msg, pubKey, sig)){
                LocalStorage.set(did, doc);
                return;
            }
        }
        }
        throw 'failed to verify signature';
    },
    deleteDID:function(did, msg, sig){
        var doc = LocalStorage.get(did);
        if (doc === null){
            throw 'did does not exist';
        }
        if (!("verificationMethod" in doc)){
            throw 'document has no verification methods'
        }
        for (var i = 0; i < doc.verificationMethod.length; i++){
            if ("publicKeyHex" in doc.verificationMethod[i]){
            var pubKey = doc.verificationMethod[i].publicKeyHex;
            if (Crypto.verifySig(msg, pubKey, sig)){
                LocalStorage.del(did, doc);
                return;
            }
        }
        }
        throw 'failed to verify signature';
    },
    createSchema:function(schema){
        if (!("id" in schema && "type" in schema)){
            throw 'example needs id and type';
        }
        var combinedKey = schema.id + "/" + schema.type;
        LocalStorage.set(combinedKey, schema);
    },
    addVC:function(cred, msg){
        if (!("context" in cred && "type" in cred && "issuer" in cred && "id" in cred && "issuanceDate" in cred && "proof" in cred && "credentialSubject" in cred && "credentialSchema" in cred)){
            throw 'missing fields';
        }

        if (!("id" in cred.credentialSubject)){
            throw 'credentialSubject needs id field';
        }

        if (!("id" in cred.credentialSchema && "type" in cred.credentialSchema)){
            throw 'credentialSchema needs id and type fields';
        }
        var schemaKey = cred.credentialSchema.id + "/" + cred.credentialSchema.type;
        var schema = LocalStorage.get(schemaKey);
        if (schema === null){
            throw 'credentialSchema not found';
        }

        if (!MatchesSchema(cred, schema)){
            throw 'credential does not match schema';
        }

        var existingCred = LocalStorage.get(cred.id);
        if (existingCred !== null){
            throw 'id already in use';
        }

        if (!Array.isArray(cred.proof)){
            cred.proof = [cred.proof];
        }
        var issuerDoc = LocalStorage.get(cred.issuer);
        if (issuerDoc === null){
            throw 'issuer does not exist';
        }
        
        for (var i = 0; i < cred.proof.length; i++){
            if (!("type" in cred.proof[i] && "created" in cred.proof[i] && "proofPurpose" in cred.proof[i] && "verificationMethod" in cred.proof[i] && "hex" in cred.proof[i])){
                throw 'proof missing fields';
            }
            if ("verificationMethod" in issuerDoc) {
                for (var v = 0; v < issuerDoc.verificationMethod.length; v++){
                    if (Crypto.verifySig(msg, issuerDoc.verificationMethod[v].publicKeyHex, cred.proof[i].hex)){
                        delete cred.proof;
                        delete cred.credentialSchema;
                        LocalStorage.set(cred.id, cred);
                        return;
                    }
                }
            }
        }
        throw 'failed to verify';
    },
    updateVC:function(cred, msg){
        if (!("context" in cred && "type" in cred && "issuer" in cred && "id" in cred && "issuanceDate" in cred && "proof" in cred && "credentialSubject" in cred && "credentialSchema" in cred)){
            throw 'missing fields';
        }

        if (!("id" in cred.credentialSubject)){
            throw 'credentialSubject needs id field';
        }

        if (!("id" in cred.credentialSchema && "type" in cred.credentialSchema)){
            throw 'credentialSchema needs id and type fields';
        }
        var schemaKey = cred.credentialSchema.id + "/" + cred.credentialSchema.type;
        var schema = LocalStorage.get(schemaKey);
        if (schema === null){
            throw 'credentialSchema not found';
        }

        if (!MatchesSchema(cred, schema)){
            throw 'credential does not match schema';
        }

        var existingCred = LocalStorage.get(cred.id);
        if (existingCred === null){
            throw 'id does not exist';
        }

        if (!Array.isArray(cred.proof)){
            cred.proof = [cred.proof];
        }
        var issuerDoc = LocalStorage.get(cred.issuer);
        if (issuerDoc === null){
            throw 'did does not exist';
        }
        
        for (var i = 0; i < cred.proof.length; i++){
            if (!("type" in cred.proof[i] && "created" in cred.proof[i] && "proofPurpose" in cred.proof[i] && "verificationMethod" in cred.proof[i] && "hex" in cred.proof[i])){
                throw 'proof missing fields';
            }
            if ("verificationMethod" in issuerDoc) {
                for (var v = 0; v < issuerDoc.verificationMethod.length; v++){
                    if (Crypto.verifySig(msg, issuerDoc.verificationMethod[v].publicKeyHex, cred.proof[i].hex)){
                        delete cred.proof;
                        delete cred.credentialSchema;
                        LocalStorage.set(cred.id, cred);
                        return;
                    }
                }
            }
        }
        throw 'failed to verify';
    },
    deleteVC:function(credID, sig, msg){
        var cred = LocalStorage.get(credID);
        if (cred === null){
            throw 'credential does not exist';
        }
        var issuerDoc = LocalStorage.get(cred.issuer);
        if (issuerDoc === null){
            throw 'issuer does not exist';
        }
        if ("verificationMethod" in issuerDoc) {
            for (var v = 0; v < issuerDoc.verificationMethod.length; v++){
                if (Crypto.verifySig(msg, issuerDoc.verificationMethod[v].publicKeyHex, sig)){
                    LocalStorage.del(cred.id);
                    return;
                }
            }
        }

        throw 'failed to verify';
    }
};
module.exports = new VCStorage;