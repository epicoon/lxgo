// @lx:namespace lx;
class Comparator {
    static simpleCompare(...args) {
        if (args.length < 2) return true;

        for (var i=1, l=args.length; i<l; i++) {
            if (lx.Json.encode(args[0]) !== lx.Json.encode(args[i])) {
                return false;
            }
        }
        return true;
    }

    static deepCompare (...args) {
        if (args.length < 2) return true;

        if (args[0] === null) {
            for (var i=1, l=args.length; i<l; i++)
                if (args[i] !== null) return false;
            return true;
        }

        if (args[0] === undefined) {
            for (var i=1, l=args.length; i<l; i++)
                if (args[i] !== undefined) return false;
            return true;
        }

        for (var i=1, l=args.length; i<l; i++) {
            if (args[i] === null || args[i] === undefined) return false;
            if (!args[0].lxCompare(args[i])) return false;
        }
        return true;
    }

    static strongCompare (...args) {
        function compare2Objects (x, y) {
            var p;

            // remember that NaN === NaN returns false
            // and isNaN(undefined) returns true
            if (isNaN(x) && isNaN(y) && typeof x === 'number' && typeof y === 'number') {
                return true;
            }

            // Compare primitives and functions.     
            // Check if both arguments link to the same object.
            // Especially useful on the step where we compare prototypes
            if (x === y) {
                return true;
            }

            // Works in case when functions are created in constructor.
            // Comparing dates is a common scenario. Another built-ins?
            // We can even handle functions passed across iframes
            if ((typeof x === 'function' && typeof y === 'function')
                || (x instanceof Date && y instanceof Date)
                || (x instanceof RegExp && y instanceof RegExp)
                || (x instanceof String && y instanceof String)
                || (x instanceof Number && y instanceof Number)
            ) {
                return x.toString() === y.toString();
            }

            // At last checking prototypes as good as we can
            if (!(x instanceof Object && y instanceof Object)) {
                return false;
            }

            if (x.isPrototypeOf(y) || y.isPrototypeOf(x)) {
                return false;
            }

            if (x.constructor !== y.constructor) {
                return false;
            }

            if (x.prototype !== y.prototype) {
                return false;
            }

            // Check for infinitive linking loops
            if (leftChain.indexOf(x) > -1 || rightChain.indexOf(y) > -1) {
                return false;
            }

            // Quick checking of one object being a subset of another.
            // TODO cache the structure of arguments[0] for performance
            for (p in y) {
                if (y.hasOwnProperty(p) !== x.hasOwnProperty(p)) {
                    return false;
                } else if (typeof y[p] !== typeof x[p]) {
                    return false;
                }
            }

            for (p in x) {
                if (y.hasOwnProperty(p) !== x.hasOwnProperty(p)) {
                    return false;
                } else if (typeof y[p] !== typeof x[p]) {
                    return false;
                }

                switch (typeof (x[p])) {
                    case 'object':
                    case 'function':
                        leftChain.push(x);
                        rightChain.push(y);

                        if (!compare2Objects (x[p], y[p])) {
                            return false;
                        }

                        leftChain.pop();
                        rightChain.pop();
                        break;

                    default:
                        if (x[p] !== y[p]) {
                            return false;
                        }
                        break;
                }
            }

            return true;
        }

        if (args.length < 2) return true;

        var leftChain, rightChain;
        for (var i=1, l=args.length; i<l; i++) {
            leftChain = [];
            rightChain = [];

            if (!compare2Objects(arguments[0], arguments[i])) {
                return false;
            }
        }

        return true;
    }
}
