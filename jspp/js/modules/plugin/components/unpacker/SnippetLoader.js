/**
 * Assembles elements tree
 * Triggers element events
 * Created recursively for nested snippets
 */
class SnippetLoader {
	constructor(loadContext, plugin, elem, infoIndex, parentLoader = null) {
		this.loadContext = loadContext;
		this.plugin = plugin;
		this.elem = elem;
		this.info = this.loadContext.getSnippetInfo(this.plugin, infoIndex);
		this.elems = [];
		this.currentElement = 0;
		this.node = new SnippetJsNode(
			this.loadContext,
			this.plugin,
			new lx.Snippet(this.elem, this.info),
			parentLoader ? parentLoader.node : null
		);
	}

	unpack() {
		if (!this.info) return;

		if (!this.elem.isSnippet)
			this.elem.isSnippet = true;

		// Create tree of lx-elements
		this.riseTree(this.elem);

		// Apply snippet root elem settings
		if (this.info.self) this.applySelf(this.info.self);

		// Unpack the rest of content
		if (this.elems.length) this.unpackContent();

		// Call to snippets JS-code
		if (this.info.js) this.node.js = this.info.js;
	}

	applySelf(info) {
		if (info.attrs)
			for (let i in info.attrs)
				this.elem.domElem.setAttribute(i, info.attrs[i]);

		if (info.classes)
			for (let i=0, l=info.classes.len; i<l; i++)
				this.elem.domElem.addClass(info.classes[i]);

		if (info.style)
			for (let st in info.style) {
				let val = info.style[st],
					stName = st.replace(/-(\w)/g, function(str, p0) {
						return p0.toUpperCase();
					});
				this.elem.domElem.style(stName, val);
			}

		if (info.props)
			for (let prop in info.props)
				if (!(prop in this.elem))
					this.elem[prop] = info.props[prop];

		if (info.funcs)
			for (let name in info.funcs)
				if (!(name in this.elem))
					this.elem[name] = this.elem.unpackFunction(info.funcs[name]);

		this.elem.inLoad = true;
		this.elem.unpackProperties();
		delete this.elem.inLoad;
		this.elem.postUnpack();
	}

	riseTree(elem) {
		for (let i=0,l=elem.getDomElem().children.length; i<l; i++) {
			let node = elem.getDomElem().children[i];

			if (node.getAttribute('lx') === null) continue;

			let info = this.info.lx[this.currentElement++],
				namespace = info._namespace ? info._namespace : 'lx';

			let namespaceObj = lx.getNamespace(namespace);
			if (!(namespaceObj)) {
				console.error('Widget not found:', namespace + '.' + info._type);
				this.elems.push(null);
				continue;
			}

			let type = null;

			if (info._type in namespaceObj) type = namespaceObj[info._type];
			else if (namespace == 'lx') type = lx.Box;

			if (type === null) {
				console.error('Widget not found:', namespace + '.' + info._type);
				this.elems.push(null);
				continue;
			}

			let el = type.rise(node);

			for (let prop in info) {
				if (prop == '_type' || prop == '_namespace' || prop in el) continue;
				el[prop] = info[prop];
			}

			if (info.key) el.key = info.key;

			el.inLoad = true;

			this.elems.push(el);

			el.parent = elem;
			el.domElem.parent = elem;
			elem.registerChild(el);

			if (node.getAttribute('lx-plugin') || node.getAttribute('lx-snippet'))
				continue;
			this.riseTree(el);
		}
	}

	unpackContent() {
		// Unpack special properties (positionin strategies, handler-function etc.)
		for (let i=0, l=this.elems.length; i<l; i++) {
			let el = this.elems[i];
			if (!el) continue;

			el.unpackProperties();
			el.restoreLinks(this);

			// Assemble nested snippets
			let snippetKey = el.getAttribute('lx-snippet');
			if (snippetKey)
				(new SnippetLoader(this.loadContext, this.plugin, el, snippetKey, this)).unpack();

			// Assemble nested plugin
			let pluginKey = el.getAttribute('lx-plugin');
			if (pluginKey)
				this.loadContext.createPluginByKey(pluginKey, el, this.plugin);

			delete el.inLoad;
		}

		// Immediatly calls
		for (let i=0, l=this.elems.length; i<l; i++) {
			let el = this.elems[i];

			// Widget post load
			/* TODO el.immediatlyPostLoad() ? el.postLoad() : */ el.displayOnce(el.postLoad);

			// On load callbacks
			this.callOnload(el);

			// If widget is displaying 
			if (el.isDisplay()) el.trigger('displayin');
		}
	}

	callOnload(el) {
		let js = el.lxExtract('forOnload');
		if (!js) return;
		for (let i=0; i<js.len; i++) {
			let item = js[i],
				 func,
				 args=null;
			if (lx.isArray(item)) {
				func = item[0];
				args = item[1];
			} else func = item;
			func = el.unpackFunction(func);
			if (args === null) func.call(el);
			else {
				if (!lx.isArray(args)) args = [args];
				func.apply(el, args);
			}
		}
	}

	/**
	 * For [[lx.Rect::restoreLinks(loader)]]
	 */
	getWidget(renderIndex) {
		return this.elems[renderIndex];
	}

	/**
	 * For [[lx.Rect::restoreLinks(loader)]]
	 */
	getCollection(src, rules=null) {
		let result = new lx.Collection();
		if (!rules) {
			for (let i=0, l=src.len; i<l; i++) {
				let item = src[i];
				result.add(this.elems[item]);
			}
			return result;
		}

		let indexName = rules.index || 'index';
		let fields = rules.fields || {};
		for (let i=0, l=src.len; i<l; i++) {
			let item = src[i];
			let elem = this.elems[item[indexName]];
			if (lx.isArray(fields)) {
				for (let j=0, ll=fields.len; j<ll; j++) {
					let name = fields[j];
					if (item[name]) elem[name] = item[name];
				}
			} else {
				for (let fieldName in fields) {
					if (!item[fieldName]) continue;
					let fieldData = fields[fieldName];
					elem[fieldData.name] = fieldData.type == 'function'
						? elem.unpackFunction(item[fieldName])
						: item[fieldName];
				}
			}
			result.add(elem);
		}
		return result;
	}
}
