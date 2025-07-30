// @lx:namespace lx;
class TreeConverter {
	treeToJson(tree) {
		let arr = [];
		for (let i=0, l=tree.keys.len; i<l; i++) {
			let key = tree.keys[i];
			_collectJSON(tree.nodes[key], key, -1, arr);
		}

		if (arr.lxEmpty()) return '';
		return JSON.stringify(arr);
	}

	jsonToTree(str) {
		const tree = new lx.Tree();
		if (str == '') return tree;

		let arr = JSON.parse( str ),
			temp = [ tree ];
		for (let i=0, l=arr.length; i<l; i++) {
			let info = arr[i],
				br = temp[ info.root+1 ].add( info.key );

			br.data = info.data;
			br.key = info.key;
			if (info.comment) br.comment = info.comment;
			if (info.fill != undefined) br.fill = info.fill;
			temp.push(br);
		}

		return tree;
	}
}

function _collectJSON(tree, key, root, arr) {
	let index = arr.length;
	let temp = {
		root,
		data: tree.data
	};
	if (tree.comment) temp.comment = tree.comment;
	if (tree.fill) temp.fill = +tree.fill;
	if (key !== '') temp.key = key;

	arr.push(temp);
	for (let i=0, l=tree.keys.len; i<l; i++) {
		let key = tree.keys[i];
		_collectJSON(tree.nodes[key], key, index, arr);
	}
}
