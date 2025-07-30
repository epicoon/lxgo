lx.Math = {
    decChangeNotation: function(n, basis) {
        str = '';
        q = Math.floor(n / basis);
        while (q) {
            a = n % basis;
            if (a>35) a = String.fromCharCode(a+29);
            else if (a>9) a = String.fromCharCode(a+87);
            str =  a + str;
            n = q;
            q = Math.floor(n / basis);
        }
        a = n % basis;
        if (a>35) a = String.fromCharCode(a+29);
        else if (a>9) a = String.fromCharCode(a+87);
        str = a + str;
        return str;
    },

    roundToZero: function(val) {
        if (val > 0) return Math.floor(val);
        if (val < 0) return Math.ceil(val);
        return 0;
    },

    modRound: function(value, precision) {
        var precision_number = Math.pow(10, precision);
        return Math.round(value * precision_number) / precision_number;
    },

    floatEqual: function(f1, f2, precision=0.0001) {
        return ( Math.max(f1, f2) - Math.min(f1, f2) < precision );
    },

    randomInteger: function(min, max) {
        return Math.floor(min + Math.random() * (max + 1 - min));
    },

    radToGrad: function(angle, precision=4) {
        return this.modRound(angle * 180 / Math.PI, precision);
    },

    selectRandomKeys: function(arr, amt) {
        var copy = [];
        for (var i in arr) copy.push( +i );
        if (amt > arr.length) amt = arr.length;

        var result = [],
            l = arr.length - 1;
        for (var i=0; i<amt; i++) {
            var r = lx.Math.randomInteger(0, l);
            result.push( copy[r] );
            copy.splice(r, 1);
            l--;
        }
        return result;
    },

    /**
     * Bisection method, example of use for calculating the root:
     * function sqrt(val) {
     *	return lx.halfDivisionMethod(0, val, function(res, precision) {
     *		var sq = res*res;
     *		if ( Math.abs(sq - val) < precision ) return 0;
     *		if ( sq > val ) return 1;
     *		if ( sq < val ) return -1;
     *	});
     * }
     * */
    halfDivisionMethod: function(min, max, condition, precision = 0.001, attempts = 1000) {
        let result = ( max - min ) * 0.5,
            counter = 0,
            prev = result;

        while (true) {
            counter++;
            if (counter === attempts) break;
            let cond = condition(result);
            if (cond == 0) break;
            if (cond == 1) max = result;
            else if (cond == -1) min = result;
            result = min + ( max - min ) * 0.5;
            if (Math.abs(prev - result) <= precision) break;
            prev = result;
        }

        return result;
    },

    parseToCalculate: function(str) {
        function calc(op0, op1, opr) {
            switch (opr) {
                case '+': return op0 + op1;
                case '-': return op0 - op1;
                case '/': return op0 / op1;
                case '*': return op0 * op1;
            }
        }
        function simpleCalc(str) {
            var nums = str.split( /[\*\/+-]/ ),
                opers = str.match( /[\*\/+-]/g );
            function applyOperation(i, op) {
                var num = calc(parseFloat(nums[i]), parseFloat(nums[i+1]), op);
                nums[i] = num; nums.splice(i+1, 1); opers.splice(i, 1); i--;
            }
            var oprPrioritet = ['*', '/'];
            for (var i=0; i<opers.length; i++) {
                if (oprPrioritet.indexOf(opers[i]) === -1) continue;
                applyOperation(i, opers[i]);
            }
            for (var i=0; i<opers.length; i++) {
                applyOperation(i, opers[i]);
            }
            return nums[0];
        }

        if ( !str.length ) return 0;
        if ( lx.isNumber(str) ) return +str;
        if ( str[0] != '=' ) return NaN;
        if ( str.length == 1 ) return 0;
        str = str.replace('=', '');
        str = str.replace(/ /g, '');
        if ( lx.isNumber(str) ) return +str;

        var simp = str.split('(');
        for (var i=simp.length-1; i>0; i--) {
            var s = simp[i],
                close = s.match( /\)/ ),
                inner = s.substring(0, close.index);
            simp[i - 1] += simpleCalc(inner) + s.substring(close.index+1, s.length);
        }
        return simpleCalc(simp[0]);
    }
};
