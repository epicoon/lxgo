// @lx:namespace lx;
class TagRenderer {
    constructor(config) {
        this.tag = config.tag || 'div';
        this.id = config.id || null;
        this.name = config.name || null;
        this.attributes = config.attributes || {};
        this.classList = config.classList || [];
        this.style = config.style || {};
        this.content = config.content || '';
        this.data = config.data || {};
    }

    static renderHtml(func) {
        let temp = new lx.Box({parent:null});
        temp.begin();
        func();
        temp.end();
        return temp.renderHtml();
    }

    toString() {
        return this.getOpenString()
            + this.getContentString()
            + this.getCloseString();
    }

    getOpenString() {
        var result = '<' + this.tag;

        if (this.id) {
            result += ' id="' + this.id + '"';
        }

        if (this.name) {
            result += ' name="' + this.name + '"';
        }

        for (var name in this.attributes) {
            result += ' ' + name + (
                (this.attributes[name] === null || this.attributes[name] === '')
                    ? ''
                    : '="' + this.attributes[name] + '"'
            );
        }

        if (this.classList.len) {
            let classList = lx.app.cssManager.defineCssClassNames(this, this.classList);
            result += ' class="' + classList.join(' ') + '"';
        }

        if (!this.style.lxEmpty()) {
            result += ' style="';
            for (var name in this.style) {
                var val = this.style[name];
                if (val === undefined || val === null || val === '') continue;
                var propName = name.replace(/([A-Z])/g, function(x){return "-" + x.toLowerCase()});
                result += propName + ':' + val + ';';
            }
            result += '"';
        }

        if (!this.data.lxEmpty()) {
            for (let name in this.data) {
                let value = this.data[name];
                result += ' data-' + name + '="' + value + '"';
            }
        }

        return result + '>';
    }

    getContentString() {
        if (this.content === undefined || this.content === null) return '';
        if (lx.isString(this.content) || lx.isNumber(this.content)) return this.content;
        if (lx.isBoolean(this.content)) {
            if (this.content) return 'true';
            return 'false';
        }
        if (lx.isInstance(this.content, lx.TagRenderer)) return this.content.toString();

        var result = '';
        if (lx.isArray(this.content)) {
            function renderArray(arr) {
                var result = '';
                for (var i=0, l=arr.len; i<l; i++) {
                    var item = arr[i];
                    if (lx.isString(item)) result += item;
                    else if (lx.isInstance(item, lx.TagRenderer)) result += item.toString();
                    else if (lx.isArray(item)) result += renderArray(item);
                }
                return result;
            }
            result += renderArray(this.content);
        }

        return result;
    }

    getCloseString() {
        return '</' + this.tag + '>';
    }
}
