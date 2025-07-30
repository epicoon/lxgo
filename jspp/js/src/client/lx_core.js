lx.entryElement = null;

lx.on = function(eventName, func) {
    lx.app.domEvent.add( document, eventName, func );
};

lx.off = function(eventName, func) {
    lx.app.domEvent.remove( document, eventName, func );
};

lx.preventDefault = function(event) {
    event = event || window.event;
    event.preventDefault ?
        event.preventDefault():
        event.returnValue = false;
};

lx.log = function(...args) {
    console.log.apply(null, args);
};

lx.logError = function(...args) {
    console.error.apply(null, args);
};


lx.timeCheck = function(msg, forse=false) {
    if (!forse && !lx.timeCheck.status) return;
    msg = msg ? (msg+'>>> ') : '';
    var time = new Date().getTime(),
        last = lx.timeCheck.last;
    lx.timeCheck.last = time;
    if (!last) return;
    console.log( msg + 'duration: ' + (time - last) + ' ms' );
};

lx.timeCheck.last = 0;
lx.timeCheck.status = true;

lx.timeCheck.start = function() {
    lx.timeCheck.last = new Date().getTime();
};

lx.timeCheck.condition = function(bool = true) {
    lx.timeCheck.status = bool;
    if (bool) lx.timeCheck.last = new Date().getTime();
};


function getEnvInfo() {
    var ua = navigator.userAgent;

    function bName() {
        if (ua.search(/Edge/) > -1) return 'edge';
        if (ua.search(/MSIE/) > -1) return 'ie';
        if (ua.search(/Trident/) > -1) return 'ie11';
        if (ua.search(/Firefox/) > -1) return 'firefox';
        if (ua.search(/Opera/) > -1) return 'opera';
        if (ua.search(/OPR/) > -1) return 'operaWebkit';
        if (ua.search(/YaBrowser/) > -1) return 'yabrowser';
        if (ua.search(/Chrome/) > -1) return 'chrome';
        if (ua.search(/Safari/) > -1) return 'safari';
        if (ua.search(/Maxthon/) > -1) return 'maxthon';
    };
    bName = bName();

    var version;
    switch (bName) {
        case 'edge':
            version = (ua.split('Edge')[1]).split('/')[1];
            break;
        case 'ie':
            version = (ua.split('MSIE ')[1]).split(';')[0];
            break;
        case 'ie11':
            bName = 'ie';
            version = (ua.split('; rv:')[1]).split(')')[0];
            break;
        case 'firefox':
            version = ua.split('Firefox/')[1];
            break;
        case 'opera':
            version = ua.split('Version/')[1];
            break;
        case 'operaWebkit':
            bName = 'opera';
            version = ua.split('OPR/')[1];
            break;
        case 'yabrowser':
            version = (ua.split('YaBrowser/')[1]).split(' ')[0];
            break;
        case 'chrome':
            version = (ua.split('Chrome/')[1]).split(' ')[0];
            break;
        case 'safari':
            version = (ua.split('Version/')[1]).split(' ')[0];
            break;
        case 'maxthon':
            version = ua.split('Maxthon/')[1];
            break;
    };

    var platform = 'desktop';

    if (/iphone|ipad|ipod|android|blackberry|mini|windows\sce|palm/i.test(navigator.userAgent.toLowerCase())) platform = 'mobile';

    var browsrObj;

    try {
        browsrObj = {
            platform: platform,
            browser: bName,
            versionFull: version,
            versionShort: version.split('.')[0]
        };
    } catch (err) {
        browsrObj = {
            platform: platform,
            browser: 'unknown',
            versionFull: 'unknown',
            versionShort: 'unknown'
        };
    };

    return browsrObj;
};
let env = getEnvInfo();
Object.defineProperty(lx, "environment", {
    get: function() { return env; }
});


lx.Json = {
    decode: function(str) {
        // Чтобы можно было парсить многострочники и табы
        var caret = String.fromCharCode(92) + String.fromCharCode(110),
            tab = String.fromCharCode(92) + String.fromCharCode(116);
        str = str.replace(/\n/g, caret);
        str = str.replace(/\t/g, tab);

        try {
            return JSON.parse(str);
        } catch (e) {
            var exp = str;
            while (true) {
                attempt = eliminateParseProblem(exp);
                if (attempt === false) throw e;
                if (attempt.success) return attempt.result;
                exp = attempt.string;
            }
        }
    },
    parse: function(str) {return this.decode(str);},

    /**
     * У JS есть косяк - он {i:1} такое правильно упакует, а такое [i:1] нет -
     * - содержимое ассоциативного массива будет проигнорировано
     * */
    encode: function(data) {
        var result = {};
        function rec(from, to) {
            for (var i in from) {
                var item = from[i];
                if (item === null || item === undefined) {
                    to[i] = null;
                } else if (lx.isArray(item)) {
                    to[i] = [];
                    rec(item, to[i]);
                } else if (lx.isObject(item)) {
                    to[i] = {};
                    rec(item, to[i]);
                } else to[i] = from[i];
            }
        }
        if (lx.isArray(data) || lx.isObject(data)) rec(data, result);
        else result = data;
        return JSON.stringify(result);
    },
    stringify: function(data) {return this.encode(data);}
};

function eliminateParseProblem(str) {
    try {
        var result = JSON.parse(str);
        return {success: true, result};
    } catch (e) {
        var i = e.message.match(/Unexpected (?:token|number) .+?(\d+)$/);
        if (i === null) {
            return false;
        } else i = +i[1];
        var pre = str.substring(0, i),
            post = str.substring(i);

        // Проблема неэкранированного экрана
        if (pre[i - 1] == '\\') {
            pre += '\\';
            return {
                success: false,
                string: pre + post
            };
        }

        // Проблема неэкранированной двойной кавычки
        pre = pre.replace(/"([^"]*)$/, String.fromCharCode(92) + '"$1');
        return {
            success: false,
            string: pre + post
        };
    }
}
