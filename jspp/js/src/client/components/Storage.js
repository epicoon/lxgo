let useCache = false,
    cache = {},
    sessionCache = {};

// @lx:namespace lx;
class Storage extends lx.AppComponent {
    useCache(bool = true) {
        useCache = bool;
    }

    get(key) {
        if (!useCache) return __get(key);
        if (!(key in cache)) cache[key] = __get(key);
        return cache[key];
    }

    set(key, value) {
        if (useCache) cache[key] = value;
        __set(key, value);
    }

    remove(key) {
        localStorage.removeItem(key);
    }

    clear() {
        localStorage.clear();
    }

    sessionGet(key) {
        if (!useCache) return  __sessionGet(key);
        if (!(key in sessionCache)) sessionCache[key] = __sessionGet(key);
        return sessionCache[key];
    }

    sessionSet(key, value) {
        if (useCache) sessionCache[key] = value;
        __sessionSet(key, value);
    }

    sessionRemove(key) {
        sessionStorage.removeItem(key);
    }

    sessionClear() {
        sessionStorage.clear();
    }
}

function __get(key) {
    var val = localStorage.getItem(key);
    if (val !== undefined && val !== null) val = lx.Json.decode(val);
    return val;
}

function __set(key, value) {
    try {
        localStorage.setItem(key, lx.Json.encode(value));
    } catch (e) {
        console.error('Local storage error');
        console.log(e);
    }
}

function __sessionGet(key) {
    var val = sessionStorage.getItem(key);
    if (val !== undefined && val !== null) val = lx.Json.decode(val);
    return val;
}

function __sessionSet(key, value) {
    try {
        sessionStorage.setItem(key, lx.Json.encode(value));
    } catch (e) {
        if (e == QUOTA_EXCEEDED_ERR) {
            console.log('sessionStorage is full');
        }
    }
}
