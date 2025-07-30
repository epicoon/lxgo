// @lx:namespace lx;
class RecursiveTreeConverter {
    treeToJson(tree) {
        let map = {};
        for (let i in tree.common.map)
            map[i] = {
                parents: tree.common.map[i].parents,
                children: tree.common.map[i].children,
                data: tree.common.map[i].data
            };

        let obj = { title:tree.key, keyCounter: tree.common.keyCounter, map };
        return JSON.stringify(obj);
    }

    jsonToTree(str) {
        let obj = JSON.parse(str);
        return this.objectToTree(obj);
    }

    objectToTree(obj) {
        let common = {
            keyCounter: obj.keyCounter,
            nodes: [],
            map: {}
        };

        for (let i in obj.map) {
            let data = obj.map[i],
                node = new lx.RecursiveTree(false, i);
            node.parents = data.parents;
            node.children = data.children;
            node.data = data.data;
            node.common = common;
            common.nodes.push(node);
            common.map[i] = node;
        }

        return common.map[obj.title];
    }
}
