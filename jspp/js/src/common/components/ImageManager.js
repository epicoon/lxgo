// @lx:namespace lx;
class ImageManager extends lx.AppComponentSettable {
    /**
     * @returns {string}
     */
    defaultSettingKey() {
        return 'imagePaths';
    }

    processSettings() {
        if (!lx.isArray(this.settings.imagePaths)) {
            this.settings.imagePaths = {
                default: this.settings.imagePaths
            };
        }
    }

    /**
     * @param {lx.Rect} ctx
     * @param {string} name
     * @returns {string}
     */
    getPath(ctx, name) {
        if (name[0] == '/') return name;

        let path;
        if (ctx.imagePaths)
            path = resolveImage(name, ctx.imagePaths);
        if (path) return path;

        if (this.app.pluginManager) {
            const plugin = this.app.pluginManager.getPlugin(ctx);
            if (plugin && plugin.imagePaths)
                path = resolveImage(name, plugin.imagePaths);
            if (path) return path;
        }

        if (this.settings.imagePaths)
            path = resolveImage(name, this.settings.imagePaths);
        if (path) return path;

        return null;
    }
}

/**
 * @param {string} name 
 * @param {map[string]string} map 
 * @returns {string}
 */
function resolveImage(name, map) {
    if (name[0] == '/') return name;

    if (name[0] != '@') {
        if (!map['default']) return null;
        return map['default'] + '/' + name;
    }

    let arr = name.match(/^@([^\/]+?)(\/.+)$/);
    if (!arr || !map[arr[1]]) return null;

    let url = '';
    if (lx.app.getProxy())
        url = lx.app.getProxy();
    url += map[arr[1]] + arr[2];
    return url;
}
