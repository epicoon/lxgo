let __isGuest = true;
let __publicFieldNames = [];
let __publicFields = {};

// @lx:namespace lx;
class User extends lx.AppComponent {
    isGuest() {
        return __isGuest;
    }

    getFieldNames() {
        return __publicFieldNames.lxClone();
    }

    getFields() {
        return __publicFields.lxClone();
    }

    set(data) {
        if (__publicFieldNames.len) return;
        
        for (let key in data) {
            __publicFieldNames.push(key);
            __publicFields[key] = data[key];
            
            Object.defineProperty(this, key, {
                get: function() {
                    return __publicFields[key];
                }
            });
        }
    }

    setGuestFlag(flag) {
        __isGuest = flag;
    }
}
