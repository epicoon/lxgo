// @lx:namespace lx;
class PresetFieldtHolder {
    /**
     * @param {string|Array<string>} valName
     * @param {any|Array<any>} defaultVal
     * @param {Function} [modifier]
     */
    constructor(valName, defaultVal, modifier) {
        this.names = lx.isArray(valName) ? valName : [valName];
        this.defaultVals = lx.isArray(defaultVal) ? defaultVal : [defaultVal];
        this.modifier = modifier;
    }

    /**
     * @param {lx.CssPreset|null} preset
     * @returns {any}
     */
    getValue(preset) {
        let vals = [];
        for (let i in this.names) {
            let name = this.names[i];
            vals.push((preset && name in preset)
                ? preset[name]
                : this.defaultVals[i]
            );
        }
        if (!this.modifier) return vals[0];
        return this.modifier.apply(null, vals);
    }
}
