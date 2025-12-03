const _list = {};
const _scopes = {};

/**
 * Initialisation:
 * - short
 * lx.app.setupComponents({
 *     cssManager: {
 *         'default': lx.CssPresetDark,
 *         'white': lx.CssPresetWhite
 *     },
 * });
 * - full
 * lx.app.setupComponents({
 *     cssManager: {
 *         defaultPreset: 'dark',
 *         scopes: {
 *             'dark': lx.CssPresetDark,
 *             'white': lx.CssPresetWhite
 *         },
 *     },
 * });
 * 
 * Setting up by code:
 * lx.app.cssManager.createPresetScope(lx.CssPresetDark);
 * lx.app.cssManager.createPresetScope(lx.CssPresetWhite, 'white');
 */

// @lx:namespace lx;
class CssManager extends lx.AppComponentSettable {
    init() {
        this.presets = new PresetsList();
        this.scopes = new ScopesList();
    }

    // @lx:<context SERVER:
    /**
     * @returns {Array<Object: {
     *     prefix {String},
     *     preset {String},
     *     elems {Array<String>},
     *     classes {Array<String>},
     *     css {String}
     * }>}
     */
    pack() {
        let res = [];
        for (let i in _scopes) {
            let scope = _scopes[i];
            res.push({
                prefix: scope.getPrefix(),
                preset: scope.context.preset.lxFullClassName(),
                elems: scope.elems,
                classes: scope.classes,
                css: scope.css
            });
        }
        return res;
    }
    // @lx:context>

    onReady() {
        // @lx:<context CLIENT:
        if (lx.app.getSetting('csrs') === 'server') {
            let css = lx.app._css;
            delete lx.app._css;
            for (let i in css) {
                let iCss = css[i],
                    scope = this.createPresetScope(iCss.preset, iCss.prefix);
                scope.unpack(iCss);
            }
        }
        // @lx:context>
    }

    /**
     * @returns {String}
     */
    defaultSettingKey() {
        return 'scopes';
    }

    processSettings() {
        if (this.settings.scopes && this.settings.scopes['default']) {
            this.settings.scopes[''] = this.settings.scopes['default'];
            delete this.settings.scopes['default'];
        }
    }

    /**
     * TODO ?deprecated?
     * @returns {boolean}
     */
    isBuilt() {
        if (!('assetBuildType' in this.settings))
            return false;
        return this.settings.assetBuildType != 'none';
    }

    /**
     * @param {Function<lx.CssPreset>|lx.CssPreset|string} preset
     * @param {String} prefix
     * @returns {lx.CssScope|null}
     */
    createPresetScope(preset, prefix = '') {
        if (lx.isString(preset))
            preset = lx.getClassConstructor(preset);
        if (lx.isFunction(preset))
            preset = new preset();
        if (!(preset instanceof lx.CssPreset)) {
            lx.logError('Constructor is not preset');
            return null;
        }

        const isDefault = (prefix === ''),
            presetKey = isDefault ? preset.lxFullClassName() : prefix;

        if (this.presets.has(presetKey)) {
            let ePreset = this.presets.get(presetKey);
            if (ePreset.preset.lxFullClassName() !== preset.lxFullClassName()) {
                lx.logError('Preset "' + prefix + '" already exists');
                return;
            }
            preset = ePreset;
        } else this.registerPreset(presetKey, preset, isDefault);

        return this.registerScope(prefix, preset);
    }

    /**
     * @param {String} name
     * @param {Function<lx.CssPreset>|lx.CssPreset} preset
     * @param {boolean} isDefault
     */
    registerPreset(name, preset, isDefault) {
        if (lx.isFunction(preset))
            preset = new preset();
        if (preset instanceof lx.CssPreset) {
            this.presets.register(name, preset);
            if (isDefault)
                this.setDefaultPreset(name);
        }
    }

    /**
     * @param {String} name
     * @param {Object} params
     */
    updatePreset(name, params) {
        if (lx.isObject(name)) {
            params = name;
            name = null;
        }

        const preset = this.getPreset(name);
        preset.update(params);
        for (let i in _scopes) {
            if (_scopes[i].context.preset !== preset) continue;
            _scopes[i].update();
        }
    }

    /**
     * @param {String} name
     */
    setDefaultPreset(name) {
        if (this.settings.defaultPreset) {
            lx.logError('Default preset is already set');
            return;
        }
        this.settings.defaultPreset = name;
    }

    /**
     * @param {String|lx.CssPreset|Function<lx.CssPreset>} name
     * @returns {lx.CssPreset|null}
     */
    getPreset(name = null) {
        if (name === null || name === '')
            return this.presets.get(this.getPresetName());

        if (lx.isString(name))
            return this.presets.get(name);

        let preset;
        if (lx.isFunction(name))
            preset = new name();
        if (preset instanceof lx.CssPreset)
            return preset;

        return null;
    }

    /**
     * @returns {String}
     */
    getPresetName() {
        return this.settings.defaultPreset || '';
    }

    /**
     * @returns {Array<String>}
     */
    getScopeNames() {
        let map = this.scopes.lxClone();
        if (this.settings.scopes)
            map.lxMerge(this.settings.scopes);
        return Object.keys(map);
    }

    /**
     * @param {String} name
     * @param {String|lx.CssPreset|Function<lx.CssPreset>} [preset]
     * @returns {lx.CssScope|null}
     */
    registerScope(name, preset = null) {
        if (this.scopes.has(name)) {
            lx.logError('Scope "' + name + '" already exists');
            return null;
        }
        return this.scopes.register(name, preset);
    }

    /**
     * @param {String} name
     * @returns {lx.CssScope|null}
     */
    getScope(name = '') {
        if (!this.scopes.has(name)) {
            if (!this.settings.scopes || !(name in this.settings.scopes))
                return null;
            this.createPresetScope(this.settings.scopes[name], name);
        }
        return this.scopes.get(name);
    }

    addElement(elem, scopeName = null) {
        if (scopeName === null) {
            let scopeNames = this.getScopeNames();
            for (let i in scopeNames)
                this.addElement(elem, scopeNames[i]);
            return;
        }
        const scope = this.getScope(scopeName);
        scope.addElement(elem);
    }

    addElements(elems, scopeName = null) {
        for (let i in elems)
            this.addElement(elems[i], scopeName);
    }

    defineCssClassNames(context, names) {
        let scopeName = (context && context.lxHasMethod('getCssScope'))
            ? context.getCssScope()
            : null;
        if (scopeName === null && this.app.pluginManager) {
            const plugin = this.app.pluginManager.getPlugin(context);
            if (plugin) scopeName = plugin.getCssScope();
        }
        if (scopeName === null) return names;

        const scope = this.getScope(scopeName);
        if (!scope)
            return names;

        if (context instanceof lx.Element)
            scope.addElement(context.constructor);

        if (scope.context.prefix == '')
            return names;

        let result = [];
        names.forEach(name=>{
            if (name == '') return;
            if (scope.hasClass(name))
                result.push(scope.context.prefix + '-' + name)
            else result.push(name);
        });
        return result;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

class PresetsList {
    /**
     * @param {String} name
     * @param {lx.CssPreset} preset
     */
    register(name, preset) {
        _list[name] = preset;
    }

    /**
     * @param {String} name
     * @returns {Boolean}
     */
    has(name) {
        return name in _list;
    }

    /**
     * @param {String} name
     * @returns {lx.CssPreset}
     */
    get(name) {
        return _list[name] || null;
    }

    /**
     * @returns {Dict<lx.CssPreset>}
     */
    getAll() {
        return _list;
    }
}

class ScopesList {
    /**
     * @param {String} name
     * @param {String|lx.CssPreset|Function<lx.CssPreset>} preset
     * @returns {lx.CssScope|null}
     */
    register(name, preset) {
        if (this.has(name)) return null;
        _scopes[name] = new lx.CssScope(name, preset);
        return _scopes[name];
    }

    /**
     * @param {String} name
     * @returns {lx.CssScope|null}
     */
    get(name) {
        if (!this.has(name)) return null;
        return _scopes[name];
    }

    /**
     * @param {String} name
     * @returns {boolean}
     */
    has(name) {
        return name in _scopes;
    }

    /**
     * @returns {Dict<lx.CssScope>}
     */
    getAll() {
        return _scopes;
    }
}
