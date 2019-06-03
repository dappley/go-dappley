Function.prototype.toString = function () {
    return "";
};

const require = (function (global) {
    var PathRegexForNotLibFile = /^\.{0,4}\//;
    var modules = new Map();

    var Module = function (id, parent) {
        this.exports = {};
        Object.defineProperty(this, "id", {
            enumerable: false,
            configurable: false,
            writable: false,
            value: id
        });

        if (parent && !(parent instanceof Module)) {
            throw new Error("parent parameter of Module construction must be instance of Module or null.");
        }
    };

    Module.prototype = {
        _load: function () {
            var $this = this,
            native_req_func = _native_require(this.id),
            temp_global = Object.create(global);
            native_req_func.call(temp_global, this.exports, this, curry(require_func, $this));
        },
        _resolve: function (id) {
            var paths = this.id.split("/");
            paths.pop();

            if (!PathRegexForNotLibFile.test(id)) {
                id = "jslib/" + id;
                paths = [];
            }

            for (const p of id.split("/")) {
                if (p == "" || p == ".") {
                    continue;
                } else if (p == ".." && paths.length > 0) {
                    paths.pop();
                } else {
                    paths.push(p);
                }
            }

            if (paths.length > 0 && paths[0] == "") {
                paths.shift();
            }

            return paths.join("/");
        },
    };

    var globalModule = new Module("main.js");
    modules.set(globalModule.id, globalModule);

    function require_func(parent, id) {
        id = parent._resolve(id);
        var module = modules.get(id);
        if (!module || !(module instanceof Module)) {
            module = new Module(id, parent);
            module._load();
            modules.set(id, module);
        }
        return module.exports;
    };

    function curry(uncurried) {
        var parameters = Array.prototype.slice.call(arguments, 1);
        var f = function () {
            return uncurried.apply(this, parameters.concat(
                Array.prototype.slice.call(arguments, 0)
            ));
        };
        Object.defineProperty(f, "main", {
            enumerable: true,
            configurable: false,
            writable: false,
            value: globalModule,
        });
        return f;
    };

    return curry(require_func, globalModule);
})(this);

const GlobalVars = {};

// const console = require('console.js');
// const ContractStorage = require('storage.js');
// const LocalContractStorage = ContractStorage.lcs;
// const GlobalContractStorage = ContractStorage.gcs;
// const BigNumber = require('bignumber.js');
const Blockchain = require('blockchain.js');
GlobalVars.Blockchain = Blockchain;
// const Event = require('event.js');

// var Date = require('date.js');
// Math.random = require('random.js');