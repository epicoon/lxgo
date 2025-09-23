class LoadContext {
	constructor() {
		this.pluginsInfo = null;
		this.rootKey = '';
		this.snippetTrees = {};
		this.snippetNodesList = {};
	}

	run(info, el, parent, callback) {
		this.pluginsInfo = info.lx;
		this.rootKey = info.root;
		for (let i in info.lx) info.lx[i].conf.key = i;

		if (info.html) {
			const elem = el.getContainer();
			elem.domElem.html(info.html);
		}

		// No need to load assets
		if (!info.assets || info.assets.lxEmpty()) {
			this.process(el, parent, callback);
			return;
		}

		this.pluginsInfo[this.rootKey].conf.dep = info.assets;

		// Syncronize assets loading and plugin start
		let synchronizer = new lx.RequestSynchronizer();

		// Register modules load request
		let modulesRequest = lx.app.dependencies.promiseModules({
			modules: info.assets.modules,
			immediately: false,
			depend: true
		});
		if (modulesRequest) synchronizer.register(modulesRequest);

		// Register scripts load requests
		let jsRequests = lx.app.dependencies.promiseScripts({
			scripts: info.assets.scripts,
			immediately: false,
			depend: true
		});
		jsRequests.forEach(r => synchronizer.register(r));

		// Register css load requests
		let cssRequests = lx.app.dependencies.promiseCss({
			css: info.assets.css,
			immediately: false,
			depend: true
		});
		cssRequests.forEach(r => synchronizer.register(r));

		// Plugin start after load assets will had finished
		synchronizer.send().then(()=>{
			this.process(el, parent, callback)
		});
	}

	process(el, parent, callback) {
		this.createPlugin(this.pluginsInfo[this.rootKey], el, parent);
		callback();
	}

	getSnippetInfo(plugin, key) {
		return this.pluginsInfo[plugin.key].snippets[key];
	}

	createPlugin(pluginInfo, el, parent) {
		// Create plugin instance
		if (!el) {
			lx.app.root.key = 'body';
			lx.app.root.clientRender();
			el = lx.app.root;
		}

		let config = pluginInfo.conf;
		if (parent) config.parent = parent;
		config.root = el;

		const plugin = lx.app.functionHelper.createAndCallFunctionWithArguments({config}, pluginInfo.js);

		// Run js-code before plugin render
		plugin.beforeRender();

		// Render snippets
		(new SnippetLoader(this, plugin, el.getContainer(), config.rsk)).unpack();

		// Note screen mode
		if (plugin.screenMode !== undefined) {
			plugin.screenMode = plugin.identifyScreenMode();
		}

		// OnLoad
		if (config.onLoad) config.onLoad.forEach(f => {
			f = lx.app.functionHelper.stringToFunction(f);
			f.call(plugin, plugin);
		});

		// Run js-code after plugin render but before run
		plugin.beforeRun();

		// Run plugin main js-code
		plugin.root.begin();
		plugin.run();

		// Build snippets code
		let code = 'const $plugin=lx.app.pluginManager.get("' + plugin.key + '");const __plugin__=Plugin;const $snippet=$plugin.root.snippet;',
			node = this.snippetTrees[plugin.key],
			snippetsJs = node.compileCode();
		code += snippetsJs[1];

		lx.app.functionHelper.createAndCallFunction(snippetsJs[0], code, null, snippetsJs[2]);

		plugin.root.end();
	}

	createPluginByKey(key, el, parentPlugin) {
		this.createPlugin(this.pluginsInfo[key], el, parentPlugin);
	}
}
