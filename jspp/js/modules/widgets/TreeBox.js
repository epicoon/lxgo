// @lx:module lx.TreeBox;

// @lx:use lx.Input;
// @lx:use lx.BasicCssContext;

/**
 * @widget lx.TreeBox
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-TreeBox
 * - lx-TW-Button
 * - lx-TW-Button-closed
 * - lx-TW-Button-opened
 * - lx-TW-Button-empty
 * - lx-TW-Button-add
 * - lx-TW-Button-del
 * - lx-TW-Label
 * 
 * Events:
 * - leafOpening
 * - leafOpened
 * - leafClosed
 * - beforeAddLeaf
 * - afterAddLeaf
 * - beforeDropLeaf
 * - afterDropLeaf
 */
// @lx:namespace lx;
class TreeBox extends lx.Box {
	static initCss(css) {
		css.useExtender(lx.BasicCssContext);
		css.addClass('lx-TreeBox', {
			color: css.preset.textColor,
			overflow: 'auto',
			borderRadius: '10px'
		});
		css.inheritAbstractClass('lx-TW-Button', 'ActiveButton', {
			color: css.preset.widgetIconColor,
			backgroundColor: css.preset.checkedMainColor
		});
		css.inheritClasses({
			'lx-TW-Button-closed':
				{ '@icon': ['\\25BA', {fontSize:10, paddingBottom:'4px', paddingLeft:'2px'}] },
			'lx-TW-Button-opened':
				{ '@icon': ['\\25BC', {fontSize:10, paddingBottom:'2px'}] },
			'lx-TW-Button-add'   :
				{ '@icon': ['\\271A', {fontSize:10, paddingBottom:'0px'}] },
			'lx-TW-Button-del'   :
				{
					backgroundColor: css.preset.hotMainColor,
					'@icon': ['\\2716', {fontSize:10, paddingBottom:'0px'}] 
				},
		}, 'lx-TW-Button');
		css.inheritClass('lx-TW-Button-empty', 'Button', {
			backgroundColor: css.preset.checkedMainColor,
			cursor: 'default'
		});
		css.addClass('lx-TW-Label', {
			overflow: 'hidden',
			whiteSpace: 'nowrap',
			textOverflow: 'ellipsis',
			backgroundColor: css.preset.textBackgroundColor,
			borderRadius: css.preset.borderRadius
		});
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [tree] {lx.Tree|lx.RecursiveTree},
	 *     [indent = 10] {Number},
	 *     [step = 5] {Number},
	 *     [leafHeight = 18] {Number},
	 *     [labelWidth = 250] {Number},
	 *     [addAllowed = false] {Boolean},
	 *     [rootAddAllowed = false] {Boolean},
	 *     [leaf] {Function} (: argument - leaf {TreeLeaf} :),
	 *     [beforeAddLeaf] {Function} (: argument - node {lx.Tree|lx.RecursiveTree} :),
	 *     [beforeDropLeaf] {Function} (: argument - leaf {TreeLeaf} :)
	 * }}
	 */
	render(config) {
		super.render(config);

		this.addClass('lx-TreeBox');

		//TODO - only px for now
		this.indent = config.indent || 10;
		this.step   = config.step   || 5;
		this.leafHeight = config.leafHeight || 25;
		this.labelWidth = config.labelWidth || 250;
		this.addAllowed = config.addAllowed || false;
		this.rootAddAllowed = (config.rootAddAllowed === undefined)
			? this.addAllowed
			: config.rootAddAllowed;

		this.beforeAddLeafHandler = config.beforeAddLeaf || null;
		this.beforeDropLeafHandler = config.beforeDropLeaf || null;
		this.leafRenderer = config.leaf || null;
		this.tree = config.tree || new lx.Tree();
		this.onAddHold = false;
		this.onDelHold = false;
		this.addBreaked = false;

		let w = this.step * 2 + this.leafHeight + this.labelWidth,
			el = new lx.Box({
				parent: this,
				key: 'work',
				width: w + 'px'
			});

		new lx.Rect({
			parent: this,
			key: 'move',
			left: w + 'px',
			width: this.step + 'px',
			style: {cursor: 'ew-resize'}
		});
	}

	// @lx:<context SERVER:
	beforePack() {
		//TODO lx.RecursiveTree
		if (this.tree) this.tree = (new lx.TreeConverter).treeToJson(this.tree);
		if (this.leafRenderer)
			this.leafRenderer = this.packFunction(this.leafRenderer);
	}
	// @lx:context>

	// @lx:<context CLIENT:
	clientRender(config) {
		super.clientRender(config);

		let work = lx(this)>work,
			move = lx(this)>move;
		work.stream({
			padding: this.indent+'px',
			step: this.step+'px',
			paddingRight: '0px',
			minHeight: this.leafHeight + 'px'
		});
		work.style('overflow', 'visible');
		move.move({ yMove: false });
		move.on('move', function() { work.width(this.left('px') + 'px'); });
		move.left( this.width('px') - this.indent - this.step + 'px' );
		move.trigger('move');
		this.prepareRoot();
	}

	postUnpack(config) {
		super.postUnpack(config);

		//TODO lx.RecursiveTree
		if (this.tree && lx.isString(this.tree))
			this.tree = (new lx.TreeConverter).jsonToTree(this.tree);

		if (this.leafRenderer && lx.isString(this.leafRenderer))
			this.leafRenderer = this.unpackFunction(this.leafRenderer);
	}

	leafs() {
		if (!lx(this)>work>leaf) return new lx.Collection();
		return lx(this)>work.getAll('leaf');
	}

	leaf(i) {
		return this.leafs().at(i);
	}

	leafByNode(node) {
		let match = null;
		this.leafs().forEach(function(a) {
			if (a.node === node) {
				match = a;
				this.stop();
			}
		});
		return match;
	}

	setTree(tree, forse = false) {
		this.tree = tree;
		this.renew(forse);
		return this;
	}

	dropTree() {
		this.tree = new lx.Tree();
		lx(this)>work.clear();
	}

	/**
	 * For open tree branches are not forgotten when the page is reloaded
	 */
	setStateMemoryKey(key) {
		this.on('leafOpening', function(e) {
			let opened = this.getOpenedInfo();
			opened.push(e.leaf.index);
			lx.app.cookie.set(key, opened.join(','));
		});
		this.on('leafClosed', function(e) {
			let opened = this.getOpenedInfo();
			lx.app.cookie.set(key, opened.join(','));
		});
		// Check cookie
		let treeState = lx.app.cookie.get(key);
		if (treeState) {
			this.useOpenedInfo(treeState.split(','));
		}
	}

	renew(forse = false) {
		if (!forse && !this.isDisplay()) return;

		let scrollTop = this.domElem.param('scrollTop'),
			opened = [];
		this.leafs().forEach(a=>{
			if (lx(a)>open.opened)
				opened.push(a.node);
		});

		lx(this)>work.clear();
		this.prepareRoot();
		for (let i in opened) {
			// todo - неэффективно
			let leaf = this.leafByNode(opened[i]);
			if (!leaf) continue;
			this.openBranch(leaf);
		}
		this.domElem.param('scrollTop', scrollTop);
		return this;
	}

	/**
	 * Returns array of opened leafs idexes
	 */
	getOpenedInfo() {
		let opened = [];
		this.leafs().forEach((a, i)=> {
			if (lx(a)>open.opened)
				opened.push(i);
		});
		return opened;
	}

	/**
	 * Open leafs according to .getOpenedInfo()
	 */
	useOpenedInfo(info) {
		for (let i=0, l=info.len; i<l; i++) {
			let leaf = this.leaf(info[i]);
			if (leaf) this.openBranch(leaf);
		}
	}

	prepareRoot() {
		this.createLeafs(this.tree, 0);
		let work = lx(this)>work;

		if (this.rootAddAllowed && !work.contains('submenu')) {
			let menu = new lx.Box({parent: work, key: 'submenu', height: this.leafHeight+'px'});
			new lx.Rect({
				key: 'add',
				parent: menu,
				width: this.leafHeight+'px',
				height: '100%',
				css: 'lx-TW-Button-add',
				click: _handlerAddNode
			});
		}
	}

	createLeafs(tree, shift, before = null) {
		if (!tree || !(tree instanceof lx.Tree || tree instanceof lx.RecursiveTree)) return;

		let config = {
			parent: lx(this)>work,
			key: 'leaf',
			height: this.leafHeight + 'px'
		};
		if (before) config.before = before;

		let result = TreeLeaf.construct(tree.count(), config, {
			preBuild: (config, i) => {
				config.node = tree.getNth(i);
				return config;
			},
			postBuild: elem => {
				elem.overflow('visible');
				elem._shift = shift;
			}
		});

		return result;
	}

	openNode(node) {
		let nodes = [],
			temp = node,
			leaf = this.leafByNode(temp);

		while (!leaf) {
			temp = temp.root;
			nodes.push(temp);
			leaf = this.leafByNode(temp);
		}

		for (let i = nodes.length - 1; i >= 0; i--) {
			let node = nodes[i],
				leaf = this.leafByNode(node);
			this.openBranch(leaf);
		}
	}

	openBranch(leaf, event) {
		event = event || this.newEvent();
		event.leaf = leaf;
		let node = leaf.node;

		if (node.filled) {
			//TODO point for logic expansion - if there is no data on the front yet
			//  but the node knows that it is not empty, a request for additional data loading can be processed here
		} else this.trigger('leafOpening', event);

		if (!node.count()) return;

		this.useRenderCache();
		let leafs = this.createLeafs(node, leaf._shift + 1, leaf.nextSibling());
		leafs.forEach(a=>{
			let shift = this.step + (this.step + this.leafHeight) * (a._shift);
			lx(a)>open.left(shift + 'px');
			lx(a)>label.left(shift + this.step + this.leafHeight + 'px');
		});
		this.applyRenderCache();

		let b = lx(leaf)>open;
		b.opened = true;
		b.removeClass('lx-TW-Button-closed');
		b.addClass('lx-TW-Button-opened');
		this.trigger('leafOpened', event);
	}

	closeBranch(leaf, event) {
		event = event || this.newEvent();
		event.leaf = leaf;
		let i = leaf.index,
			shift = leaf._shift,
			next = this.leaf(++i);
		while (next && next._shift > shift) next = this.leaf(++i);

		let count = next ? next.index - leaf.index - 1 : Infinity;

		leaf.parent.del('leaf', leaf.index + 1, count);
		let b = lx(leaf)>open;
		b.opened = false;
		b.removeClass('lx-TW-Button-opened');
		b.addClass('lx-TW-Button-closed');
		this.trigger('leafClosed', event);
	}

	holdAdding() {
		this.onAddHold = true;
	}

	breakAdding() {
		this.onAddHold = false;
		this.addBreaked = true;
	}

	resumeAdding(newNodeAttributes = {}) {
		const node = this.onAddHold;
		this.onAddHold = false;
		return _addProcess(this, node, newNodeAttributes);
	}

	/**
	 * @param {lx.Tree} parentNode
	 * @param {Object} newNodeAttributes
	 */
	add(parentNode, newNodeAttributes) {
		if (parentNode.root) {
			let leaf = this.leafByNode(parentNode),
				but = lx(leaf)>open;
			but.opened = true;
		}
		let key = newNodeAttributes.key || parentNode.genKey(),
			node = parentNode.add(key);
		for (let f in newNodeAttributes)
			if (f == 'data' || !(f in node)) node[f] = newNodeAttributes[f];
		this.renew();
		return node;
	}

	holdDropping() {
		this.onDelHold = true;
	}

	breakDropping() {
		this.onDelHold = false;
	}

	resumeDropping() {
		const leaf = this.onDelHold;
		this.onDelHold = false;
		_delProcess(this, leaf);
	}
	
	drop(leaf) {
		let node = leaf.node;
		node.del();
		this.renew();
	}
	// @lx:context>

	beforeAddLeaf(func) {
		this.beforeAddLeafHandler = func;
	}

	beforeDropLeaf(func) {
		this.beforeDropLeafHandler = func;
	}

	setLeafRenderer(func) {
		this.leafRenderer = func;
	}

	setLeafsRight(val) {
		let work = lx(this)>work,
			move = lx(this)>move,
			w = this.width('px') - val;
		work.width(w + 'px');
		move.left(w + 'px');
	}

	setLeafsRightForButtons(count) {
		this.setLeafsRight(this.step * (count + 1) + this.leafHeight * count);
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context CLIENT:
class TreeLeaf extends lx.Box {
	constructor(config) {
		super(config);
		this.node = config.node;
		this.box = this.ancestor({is: lx.TreeBox});
		this.create();
	}

	create() {
		let tw = this.box,
			but = new lx.Rect({
				parent: this,
				key: 'open',
				geom: [0, 0, tw.leafHeight+'px', tw.leafHeight+'px'],
				click: _handlerToggleOpened
			}).addClass(
				(this.node.count() || (this.node.filled))
					? 'lx-TW-Button-closed' : 'lx-TW-Button-empty'
			);
		but.opened = false;

		let lbl = new lx.Box({
			parent: this,
			key: 'label',
			left: tw.leafHeight + tw.step + 'px',
			css: 'lx-TW-Label'
		});

		if ( tw.leafRenderer ) tw.leafRenderer(this);
	}

	createChild(config={}) {
		let tw = this.box;
		if (config.width === undefined) config.width = tw.leafHeight + 'px';
		if (config.height === undefined) config.height = tw.leafHeight + 'px';

		config.parent = this;
		config.right = -(tw.step + tw.leafHeight) * (this.childrenCount() - 1) + 'px';

		let type = config.widget || lx.Box;

		return new type(config);
	}

	createButton(config={}) {
		if (config instanceof Function) config = {click: config};

		if (config.key === undefined) config.key = 'button';
		if (config.widget === undefined) config.widget = lx.Box;
		return this.createChild(config);
	}

	createAddButton(config = {}) {
		let b = this.createButton(config);
		b.click(_handlerAddNode);
		if (!config.css)
			b.addClass('lx-TW-Button-add');
		return b;
	}

	createDropButton(config = {}) {
		let b = this.createButton(config);
		b.click(_handlerDelNode);
		if (!config.css)
			b.addClass('lx-TW-Button-del');
		return b;
	}
}
// @lx:context>

function _handlerToggleOpened(event) {
	let tw = this.ancestor({is: lx.TreeBox}),
		l = this.parent;
	if (this.opened) tw.closeBranch(l, event);
	else tw.openBranch(l, event);
}

function _handlerAddNode() {
	const tw = this.ancestor({is: lx.TreeBox}),
		isRootAdding = (this.key == 'add'),
		pNode = isRootAdding ? tw.tree : this.parent.node;

	let obj = null;
	if (tw.beforeAddLeafHandler)
		obj = tw.beforeAddLeafHandler.call(this, pNode);
	if (tw.addBreaked) {
		tw.addBreaked = false;
		return;
	}
	if (tw.onAddHold) {
		tw.onAddHold = pNode;
		return;
	}

	_addProcess(tw, pNode, obj || {});
}

function _handlerDelNode() {
	let leaf = this.parent,
		tw = leaf.box;

	if (tw.beforeDropLeafHandler)
		tw.beforeDropLeafHandler.call(this, leaf);
	if (tw.onDelHold) {
		tw.onDelHold = leaf;
		return;
	}

	_delProcess(tw, leaf);
}

function _addProcess(self, parentNode, newNodeAttributes = {}) {
	const e = self.newEvent({parentNode, newNodeAttributes});

	if (self.trigger(
		'beforeAddLeaf',
		e) === false
	) return null;
	
	const newNode = self.add(parentNode, newNodeAttributes);
	e.newNode = newNode;
	self.trigger('afterAddLeaf', e);
	return newNode;
}

function _delProcess(self, leaf) {
	let node = leaf.node;

	if (self.trigger(
		'beforeDropLeaf',
		self.newEvent({leaf, node})
	) === false) return;

	self.drop(leaf);

	self.trigger('afterDropLeaf');
}
