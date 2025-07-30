// @lx:namespace lx;
class CssPreset {
    constructor() {
        _init(this);
    }

    createProperty(...args) {
        return new lx.CssValue(this, '', args.join(' '));
    }

    gradient(...args) {
        let dir, color0, color1;
        if (args.length == 2) {
            dir = 'to top';
            color0 = args[0];
            color1 = args[1];
        } else {
            dir = args[0];
            color0 = args[1];
            color1 = args[2];
        }
        if (!dir.match(/^to /)) dir = 'to ' + dir;
        return new lx.CssValue(this, '', 'linear-gradient(' + dir + ',' + color0 + ',' + color1 + ')');
    }
    
    getSettings() {
        return {};
    }

    get name() {
        return lx(STATIC).getName();
    }

    static getName() {
        return null;
    }
}

function _init(self) {
    const map = self.getSettings();
    for (let name in map) {
        Object.defineProperty(self, name, {
            get: function() {
                return new lx.CssValue(self, name, map[name]);
            }
        });
    }
}

// @lx:namespace lx;
class CssValue {
    constructor(preset, name, value) {
        this.preset = preset;
        this.name = name;
        this.value = value;
    }

    toCssString() {
        if (lx.isString(this.value) || lx.isNumber(this.value))
            return this.value;

        if (lx.implementsInterface(this.value, {methods:['toCssString']}))
            return this.value.toCssString();

        return this.value;
    }

    [Symbol.toPrimitive](hint) {
        switch (hint) {
            case 'number':
            case 'string':
            default:
                return this.toCssString();
        }
    }
}
