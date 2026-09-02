// @lx:namespace lx;
class PresetFieldHolder {
    /**
     * @param {lx.CssPreset} preset
     * @param {string|Array<string>} valName
     * @param {any|Array<any>} defaultVal
     * @param {Function} [modifier]
     */
    constructor(preset, valName, defaultVal, modifier) {
        this.preset = preset;
        this.names = lx.isArray(valName) ? valName : [valName];
        this.defaultVals = lx.isArray(defaultVal) ? defaultVal : [defaultVal];
        this.modifier = modifier;
    }

    /**
     * @param {lx.CssPreset|null} preset
     * @returns {any}
     */
    getValue(preset) {
        return _getValue(this, preset);
    }

    [Symbol.toPrimitive](hint) {
        switch (hint) {
            case 'number': {
                let val = _getValue(this);
                return +val;
            } case 'string':
            default: {
                let val = _getValue(this);
                return ''+val;
            }
        }
    }
}

function _getValue(self, preset) {
    preset = preset || self.preset;
    let vals = [];
    for (let i in self.names) {
        let name = self.names[i];
        vals.push((preset && name in preset)
            ? preset[name]
            : self.defaultVals[i]
        );
    }
    if (!self.modifier) return vals[0];
    return self.modifier.apply(null, vals);
}
