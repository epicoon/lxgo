class BoxChildren {
    constructor() {
        this.data = [];
    }

    isEmpty() {
        return !this.data.length;
    }

    count() {
        return this.data.length;
    }

    push(el) {
        this.data.push(el);
    }

    remove(el) {
        this.data.lxRemove(el);
    }

    contains(el) {
        return this.data.includes(el);
    }

    indexOf(el) {
        return this.data.indexOf(el);
    }

    get(num) {
        if (num >= this.data.length) return null;
        return this.data[num];
    }
    
    getByCondition(condition) {
        for (let i=0, l=this.data.len; i<l; i++)
            if (condition(this.data[i]))
                return this.data[i];
        return null;
    }

    insertBefore(el, next) {
        var index = this.data.indexOf(next);
        if (lx.isArray(el)) this.data.apply(this.data, [index, 0].concat(el));
        else this.data.splice(index, 0, el);
    }

    prev(elem) {
        var index = this.data.indexOf(elem);
        if (index == 0) return null;
        return this.data[index - 1];
    }

    next(elem) {
        var index = this.data.indexOf(elem);
        if (index + 1 == this.data.length) return;
        return this.data[index + 1];
    }

    last() {
        if (!this.data.length) return null;
        return this.data[this.data.length - 1];
    }

    forEach(f) {
        this.data.forEach(f);
    }

    reset() {
        this.data = [];
    }
}
