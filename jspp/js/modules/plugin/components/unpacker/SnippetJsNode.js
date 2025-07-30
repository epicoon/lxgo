/**
 * Snippets tree to assemble namespaces according to hierarchy
 */
class SnippetJsNode {
	constructor(loadContext, plugin, snippet, parentNode = null) {
		this.loadContext = loadContext;
		this.elem = snippet.widget;
		this.js = null;
		this.children = [];
		this.key = 'k' + this.loadContext.snippetsCounter++;
		this.loadContext.snippetNodesList[this.key] = this;
		if (parentNode) parentNode.addChild(this);
		else this.loadContext.snippetTrees[plugin.key] = this;
	}

	isEmpty() {
		return !this.children.len;
	}

	addChild(node) {
		this.children.push(node.key);
	}

	child(i) {
		return this.loadContext.snippetNodesList[this.children[i]];
	}

	compileCode() {
		let argsStr = [],
			args = [],
			counter = 0,
			pre = '(function(_w){const Snippet=_w.snippet;Snippet.run();_w.begin();_w=undefined;',
			post = 'Snippet.widget.end();',
			begin = [],
			end = [];

		function rec(node) {
			args.push(node.elem);
			let elem = '__lxb_' + counter;
			counter++;
			argsStr.push(elem);
			let js = node.js ? node.js.replace(/([^;])$/, '$1;') : '';
			let head = pre + js + post;
			let tail = '})('+elem+');';

			if (node.isEmpty()) begin.push(head + tail);
			else {
				begin.push(head);
				end.push(tail);
			}
			for (let i=0, l=node.children.len; i<l; i++) rec(node.child(i));
		}
		rec(this);

		let code = '';
		for (let i=0, l=begin.len; i<l; i++) code += begin[i];
		for (let i=end.len-1; i>=0; i--) code += end[i];
		return [argsStr.join(','), code, args];
	}
}
