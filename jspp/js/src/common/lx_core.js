lx.LEFT = 1;
lx.CENTER = 2;
lx.WIDTH = 2;
lx.RIGHT = 3;
lx.JUSTIFY = 4;
lx.TOP = 5;
lx.MIDDLE = 6;
lx.HEIGHT = 6;
lx.BOTTOM = 7;
lx.VERTICAL = 1;
lx.HORIZONTAL = 2;

// @lx:<context CLIENT:
// @lx:require ../client/lx_core;
// @lx:context>
// @lx:<context SERVER:
// @lx:require ../server/lx_core;
// @lx:context>

/**
 * Sequence \r\n, codes 13 and 10
 */
Object.defineProperty(lx, "EOL", {
    get: function() { return String.fromCharCode(13) + String.fromCharCode(10); }
});

lx.getFirstDefined = function (...args) {
    for (var i = 0, l = args.len; i < l; i++)
        if (args[i] !== undefined) return args[i];
    return undefined;
};

lx.isInstance = function (value, constructor) {
    if (value === undefined)
        return (constructor === undefined);
    if (value === null)
        return (constructor === null);
    if (constructor === Number) return this.isNumber(value);
    if (constructor === String) return this.isString(value);
    if (this.isString(constructor)) constructor = this.getNamespace(constructor);
    if (value.constructor && value.constructor === constructor)
        return true;
    return value instanceof constructor;
};

lx.implementsInterface = function(value, _interface) {
    if (value === undefined || value === null)
        return false;

    for (let i in _interface.methods) {
        let method = _interface.methods[i];
        if (!value[method] || !this.isFunction(value[method]))
            return false;
    }

    return true;
};

lx.isNumber = function (value) {
    if (value === undefined || value === null) return false;
    if ( value.push !== undefined ) return false;
    return ( !isNaN(parseFloat(value)) && isFinite(value) );
};

lx.isBoolean = function (value) {
    if (value === undefined || value === null) return false;
    return value.constructor === Boolean;
};

lx.isString = function (value) {
    if (value === undefined || value === null) return false;
    return value.constructor === String;    
};

lx.isArray = function (value) {
    if (value === undefined || value === null) return false;
    return value.constructor === Array;
};

lx.isStrictObject = function (value) {
    if (value === undefined || value === null) return false;
    return (value.constructor === Object || value.lxClassName() == 'Object');
};

lx.isObject = function (value) {
    if (value === undefined || value === null) return false;

    return (value.constructor === Object || value.lxClassName() == 'Object'
        || (typeof value === 'object'
            && Object.prototype.toString.call(value) === '[object Object]'
        )
    );
};

lx.isDomELem = function (value) {
    if (value === undefined || value === null) return false;

    return (value.constructor === Object || value.lxClassName() == 'Object'
        || (typeof value === 'object'
            && /^\[object HTML/.test(Object.prototype.toString.call(value))
        )
    );
}

lx.isFunction = function (value) {
    if (value === undefined || value === null) return false;
    return value.constructor === Function;
};

lx.clone = function (value) {
    if (value === undefined) return undefined;
    if (value === null) return null;
    if (lx.isString(value) || lx.isNumber(value) || lx.isBoolean(value))
        return value;
    
    return value.lxClone();
};

lx.createNamespace = function (namespace, props) {
    var arr = namespace.split(/[.\\]/),
        temp = lx.globalContext;
    for (var i = 0, l = arr.length; i < l; i++) {
        if (temp[arr[i]] === undefined) temp[arr[i]] = {};
        temp = temp[arr[i]];
    }
    if (props) temp.lxMerge(props);
    return temp;
};

lx.getNamespace = function (namespace) {
    var arr = lx.isString(namespace)
        ? namespace.split(/[.\\]/)
        : (lx.isArray(namespace) ? namespace : null);
    if (!arr) return null;
    if (arr.lxEmpty()) return lx.globalContext;

    var temp = lx.globalContext;
    for (var i = 0, l = arr.length; i < l; i++) {
        if (temp[arr[i]] === undefined) return null;
        temp = temp[arr[i]];
    }
    return temp;
};

lx.getClassConstructor = function (fullClassName) {
    if (this.isFunction(fullClassName)) return fullClassName;
    var arr = fullClassName.split(/[.\\]/),
        name = arr.pop(),
        nmsp = lx.getNamespace(arr);
    if (!nmsp) return null;
    if (name in nmsp && this.isFunction(nmsp[name])) return nmsp[name];
    return null;
};

lx.classExists = function (name) {
    return !!lx.getClassConstructor(name);
};

lx.createObject = function (fullClassName, args = undefined) {
    if (this.isFunction(fullClassName)) {
        if (args === undefined) return new fullClassName();
        return new (Function.prototype.bind.apply(
            fullClassName,
            [null].lxMerge(args)
        ));
    }

    var arr = lx.isString(fullClassName)
        ? fullClassName.split(/[.\\]/)
        : (lx.isArray(fullClassName) ? fullClassName : null);
    if (!arr) return null;

    var temp = lx.globalContext;
    for (var i = 0, l = arr.length; i < l; i++) {
        if (temp[arr[i]] === undefined) return null;

        if (i == l - 1) {
            if (args !== undefined) {
                return new (Function.prototype.bind.apply(
                    temp[arr[i]],
                    [null].lxMerge(args)
                ));
            } else return new temp[arr[i]]();
        } else temp = temp[arr[i]];
    }

    return null;
};
