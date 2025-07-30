// @lx:namespace lx;
class RecursiveTree {
    constructor(tree = null, key = null) {
        this.parents = [];
        this.children = [];

        if (tree) {
            this.common = tree.common;
            this.parents.push(tree.key);
        } else if (tree !== false) {
            this.common = {
                keyCounter: 0,
                nodes: [],
                map: {}
            };
        } else {
            this.common = null;
        }

        this.key = key || this.genKey();
        if (this.common) {
            if (this.key in this.common.map)
                throw new Error('RecursiveTree: key already exists');

            this.common.nodes.push(this);
            this.common.map[this.key] = this;
        }
        this.data = {};
    }

    count() {
        return this.children.length;
    }

    getNth(i) {
        return this.common ? this.common.map[this.children[i]] : null;
    }

    genKey() {
        return 'r' + this.common.keyCounter++;
    }

    add(keyOrNode = null) {
        if (keyOrNode instanceof lx.RecursiveTree)
            return this.addNode(keyOrNode);
        return this.addNew(keyOrNode);
    }

    addNode(node) {
        if (!this.common.nodes.includes(node))
            throw new Error('RecursiveTree: the node from different tree');

        if (node.key in this.children) return node;
        this.children.push(node.key);
        node.parents.push(this.key);
    }

    addNew(key = null) {
        const node = new this.constructor(this, key);
        this.children.push(node.key);
        return node;
    }

    setData(data = {}, unique = true) {
        if (unique && (lx.isObject(data) || lx.isArray(data)))
            this.data = data.lxClone();
        else this.data = data;
    }

    del() {
        //TODO
    }
}
