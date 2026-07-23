# The package helps to work with JS

> Actual version: `v0.1.0-alpha.28`. [Details](https://github.com/epicoon/lxgo/tree/master/jspp/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)


## Provides:
* Build JS-bundle from different JS files
* Create frontend application
* Set of widgets
* CSS organization:
  - Simple CSS-preprocessor on JS
  - Presets
  - Widgets customization using CSS-scopes
* Models and binding reactivity mechanisms
* Internationalization
* Simple AJAX handling
* Way to organize code with plugins
* Markup language `lxml`
* Set of auxiliary tools


## Contents:
* [Start using](https://github.com/epicoon/lxgo/tree/master/jspp/doc/start.md)
* [Modules](https://github.com/epicoon/lxgo/tree/master/jspp/doc/modules.md)
* [Application components](https://github.com/epicoon/lxgo/tree/master/jspp/doc/components.md)
* [Elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md)
* [Widgets](https://github.com/epicoon/lxgo/tree/master/jspp/doc/widgets.md)
* [Positioning strategies](https://github.com/epicoon/lxgo/tree/master/jspp/doc/positioning-strategies.md)
* [CSS](https://github.com/epicoon/lxgo/tree/master/jspp/doc/css.md)
* [Models and binding](https://github.com/epicoon/lxgo/tree/master/jspp/doc/models.md)
* [Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md)
* [Preprocessor features](https://github.com/epicoon/lxgo/tree/master/jspp/doc/pp.md)
* [LXML](https://github.com/epicoon/lxgo/tree/master/jspp/doc/lxml.md)


## Also worth knowing
* **AJAX** — plugins get built-in AJAX routing out of the box, see [Plugins](https://github.com/epicoon/lxgo/tree/master/jspp/doc/plugins.md); for arbitrary requests from client-side code, use `lx.HttpRequest`/`lx.Request` directly (`.send()`/`.then()`/`.catch()`).
* **Auxiliary tools** — a set of small utility classes ships with the package, e.g. `lx.Timer` (frame-based periodic/delayed actions) and `lx.TableManager` (keyboard navigation and cell selection for `lx.Table`); browse `js/src/*/tools` and `js/modules/*.js` for the full set.


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
