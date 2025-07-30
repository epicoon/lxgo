/*
 * ATTENTION: The "eval" devtool has been used (maybe by default in mode: "development").
 * This devtool is neither made for production nor for readable output files.
 * It uses "eval()" calls to create a separate source file in the browser devtools.
 * If you are trying to read the output file, select a different devtool (https://webpack.js.org/configuration/devtool/)
 * or disable the default devtool with "devtool: false".
 * If you are looking for production-ready output files, see mode: "production" (https://webpack.js.org/configuration/mode/).
 */
/******/ (() => { // webpackBootstrap
/******/ 	"use strict";
/******/ 	var __webpack_modules__ = ({

/***/ "./src/App.js":
/*!********************!*\
  !*** ./src/App.js ***!
  \********************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

eval("__webpack_require__.r(__webpack_exports__);\n/* harmony import */ var _AuthManager__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./AuthManager */ \"./src/AuthManager.js\");\n\nclass App {\n  constructor() {\n    this.eventHandlers = {};\n    this.authManager = new _AuthManager__WEBPACK_IMPORTED_MODULE_0__[\"default\"](this);\n  }\n  subscribe(event, handler) {\n    if (!(event in this.eventHandlers)) this.eventHandlers[event] = [];\n    this.eventHandlers[event].push(handler);\n  }\n  trigger(event) {\n    if (!(event in this.eventHandlers)) return;\n    this.eventHandlers[event].forEach(handler => handler());\n  }\n  async fetch(url, params = {}) {\n    const success = await _prepareParams(this, params);\n    if (!success) {\n      console.error('Can not send request');\n      return null;\n    }\n    return await fetch(url, params);\n  }\n  run() {\n    this.authManager.checkTokens();\n  }\n}\nconst app = new App();\nwindow.lxAuth = {\n  TOKENS_FOUND: 'TokensFound',\n  TOKENS_NOT_FOUND: 'TokensNotFound',\n  TOKENS_REMOVED: 'TokensRemoved',\n  app: app,\n  run: function () {\n    this.app.run();\n  },\n  goToAuth: function () {\n    this.app.authManager.goToAuth();\n  },\n  logOut: function () {\n    this.app.authManager.logOut();\n  },\n  getUserData: async function () {\n    return await this.app.authManager.getUserData();\n  },\n  on: function (event, handler) {\n    this.app.subscribe(event, handler);\n  },\n  fetch: async function (url, params = {}) {\n    return await this.app.fetch(url, params);\n  }\n};\n\n/**\n * @param {App} app \n * @param {Object} params \n * @returns {bool}\n */\nasync function _prepareParams(app, params) {\n  let accessToken = app.authManager.getAccessToken();\n  if (!accessToken.isActive()) {\n    const refreshToken = app.authManager.getRefreshToken();\n    if (!refreshToken.isActive()) {\n      return false;\n    }\n    const result = await app.authManager.refreshTokens();\n    if (!result) {\n      return false;\n    }\n    accessToken = app.authManager.getAccessToken();\n  }\n  if (!('headers' in params)) params.headers = {};\n  params.headers['Authorization'] = 'Bearer ' + accessToken.value;\n  return true;\n}\n\n//# sourceURL=webpack://client/./src/App.js?");

/***/ }),

/***/ "./src/AuthManager.js":
/*!****************************!*\
  !*** ./src/AuthManager.js ***!
  \****************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export */ __webpack_require__.d(__webpack_exports__, {\n/* harmony export */   \"default\": () => (/* binding */ AuthManager)\n/* harmony export */ });\nclass AuthManager {\n  constructor(app) {\n    this.app = app;\n    this.settings = {};\n    this.accessToken = null;\n    this.refreshToken = null;\n    this.active = _loadSettings(this);\n    this.redirecting = false;\n  }\n\n  /**\n   * @returns {Token}\n   */\n  getAccessToken() {\n    return _getAccessToken(this);\n  }\n\n  /**\n   * @returns {Token}\n   */\n  getRefreshToken() {\n    return _getRefreshToken(this);\n  }\n\n  /**\n   * @trigger lxAuth.TOKENS_FOUND\n   * @trigger lxAuth.TOKENS_NOT_FOUND\n   */\n  checkTokens() {\n    if (!this.active) return;\n    const accessToken = _getAccessToken(this);\n    if (!accessToken.isActive()) {\n      const refreshToken = _getRefreshToken(this);\n      if (!refreshToken.isActive()) {\n        this.app.trigger('TokensNotFound');\n        return;\n      }\n    }\n    this.app.trigger('TokensFound');\n  }\n\n  /**\n   * @returns {bool}\n   */\n  async refreshTokens() {\n    const refreshToken = _getRefreshToken(this);\n    if (!refreshToken.isActive()) {\n      console.error(\"Refresh auth token invalid\");\n      return false;\n    }\n    const response = await fetch(this.settings.refresh_path, {\n      method: 'POST',\n      headers: {\n        'Content-Type': 'application/json'\n      },\n      body: JSON.stringify({\n        refresh_token: refreshToken.value\n      })\n    });\n    if (!response.ok) {\n      console.error(\"Refresh auth tokens failed\");\n      return false;\n    }\n    const data = await response.json();\n    const accessToken = _getAccessToken(this);\n    accessToken.init(data.access_token, data.access_token_expired);\n    accessToken.toStorage('lxAuthAccessToken');\n    refreshToken.init(data.refresh_token, data.refresh_token_expired);\n    refreshToken.toStorage('lxAuthRefreshToken');\n    return true;\n  }\n  async goToAuth() {\n    if (this.redirecting) return;\n    this.redirecting = true;\n    const state = await _genState(this);\n    if (state === null) {\n      console.error(\"Can not redirect\");\n      return;\n    }\n    const authData = this.settings;\n    _postRedirect(`${authData.server}/auth`, {\n      response_type: 'code',\n      client_id: authData.id,\n      redirect_uri: authData.redirect_uri,\n      state\n    });\n  }\n\n  /**\n   * @trigger lxAuth.TOKENS_REMOVED\n   */\n  async logOut() {\n    const refreshToken = _getRefreshToken(this);\n    if (!refreshToken.isActive()) {\n      _dropTokens(this);\n      this.app.trigger('TokensRemoved');\n      return;\n    }\n    const accessToken = _getAccessToken(this);\n    const response = await fetch(this.settings.logout_path, {\n      method: 'GET',\n      headers: {\n        'Authorization': 'Bearer ' + accessToken.value\n      }\n    });\n    if (response.ok) {\n      _dropTokens(this);\n      this.app.trigger('TokensRemoved');\n    }\n  }\n\n  /**\n   * @returns {Object: {\n   *     {bool} success,\n   *     {string} login,\n   *     {Object} data\n   * }}\n   */\n  async getUserData() {\n    const response = await this.app.fetch(this.settings.user_data_path);\n    if (!response || !response.ok) {\n      console.error('Fetching user data failed');\n      return {\n        success: false\n      };\n    }\n    const data = await response.json();\n    data.success = true;\n    return data;\n  }\n}\n\n/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *\n * PRIVATE\n * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */\n\n/**\n * @private\n * @param {AuthManager} self \n * @returns {bool}\n */\nfunction _loadSettings(self) {\n  const as = window._lxauth_settings;\n  if (!as || as === '') {\n    console.error('Auth settings are not available');\n    return false;\n  }\n  delete window._lxauth_settings;\n  let sett;\n  try {\n    sett = JSON.parse(as);\n  } catch (e) {\n    console.error('Can not parse auth settings');\n    return false;\n  }\n  self.settings = sett;\n  return true;\n}\n\n/**\n * @param {AuthManager} self \n * @returns {string}\n */\nasync function _genState(self) {\n  const response = await fetch(self.settings.state_path, {\n    method: 'POST',\n    headers: {\n      'Content-Type': 'application/json'\n    },\n    body: JSON.stringify({\n      uri: window.location.href\n    })\n  });\n  if (!response || !response.ok) {\n    console.error('Fetching state failed');\n    return null;\n  }\n  const data = await response.json();\n  return data.state;\n}\n\n/**\n * @private\n * @param {AuthManager} self \n * @returns {Token}\n */\nfunction _getAccessToken(self) {\n  if (self.accessToken === null) {\n    _readToken(self, 'accessToken', 'lxAuthAccessToken');\n  }\n  return self.accessToken;\n}\n\n/**\n * @private\n * @param {AuthManager} self \n * @returns {Token}\n */\nfunction _getRefreshToken(self) {\n  if (self.refreshToken === null) {\n    _readToken(self, 'refreshToken', 'lxAuthRefreshToken');\n  }\n  return self.refreshToken;\n}\n\n/**\n * @private\n * @param {AuthManager} self\n * @param {string} selfKey\n * @param {string} lsKey\n */\nfunction _readToken(self, selfKey, lsKey) {\n  if (!self.active) return false;\n  self[selfKey] = new Token();\n  let tokenData = localStorage.getItem(lsKey);\n  if (!tokenData) {\n    return;\n  }\n  try {\n    tokenData = JSON.parse(tokenData);\n  } catch (e) {\n    console.error(e);\n    return;\n  }\n  self[selfKey].init(tokenData[0], tokenData[1]);\n}\n\n/**\n * @private\n * @param {AuthManager} self \n */\nfunction _dropTokens(self) {\n  localStorage.removeItem('lxAuthAccessToken');\n  localStorage.removeItem('lxAuthRefreshToken');\n  self.accessToken = null;\n  self.refreshToken = null;\n}\n\n/**\n * @private\n * @param {string} url\n * @param {Object} params\n */\nfunction _postRedirect(url, params) {\n  const form = document.createElement(\"form\");\n  form.method = \"POST\";\n  form.action = url;\n  for (const key in params) {\n    if (params.hasOwnProperty(key)) {\n      const input = document.createElement(\"input\");\n      input.type = \"hidden\";\n      input.name = key;\n      input.value = params[key];\n      form.appendChild(input);\n    }\n  }\n  document.body.appendChild(form);\n  form.submit();\n}\nclass Token {\n  constructor() {\n    this.exists = false;\n    this.value = null;\n    this.expiresAt = null;\n  }\n\n  /**\n   * @param {string} value \n   * @param {number} expiresAt \n   */\n  init(value, expiresAt) {\n    this.value = value;\n    this.expiresAt = +expiresAt;\n    this.exists = true;\n  }\n\n  /**\n   * @param {string} key \n   */\n  toStorage(key) {\n    localStorage.setItem(key, '[\"' + this.value + '\", ' + this.expiresAt + ']');\n  }\n\n  /**\n   * @returns {bool}\n   */\n  isExpired() {\n    const currentTime = Math.floor(Date.now() / 1000);\n    return this.expiresAt <= currentTime;\n  }\n\n  /**\n   * @returns {bool}\n   */\n  isActive() {\n    return this.exists && !this.isExpired();\n  }\n}\n\n//# sourceURL=webpack://client/./src/AuthManager.js?");

/***/ })

/******/ 	});
/************************************************************************/
/******/ 	// The module cache
/******/ 	var __webpack_module_cache__ = {};
/******/ 	
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/ 		// Check if module is in cache
/******/ 		var cachedModule = __webpack_module_cache__[moduleId];
/******/ 		if (cachedModule !== undefined) {
/******/ 			return cachedModule.exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = __webpack_module_cache__[moduleId] = {
/******/ 			// no module.id needed
/******/ 			// no module.loaded needed
/******/ 			exports: {}
/******/ 		};
/******/ 	
/******/ 		// Execute the module function
/******/ 		__webpack_modules__[moduleId](module, module.exports, __webpack_require__);
/******/ 	
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/ 	
/************************************************************************/
/******/ 	/* webpack/runtime/define property getters */
/******/ 	(() => {
/******/ 		// define getter functions for harmony exports
/******/ 		__webpack_require__.d = (exports, definition) => {
/******/ 			for(var key in definition) {
/******/ 				if(__webpack_require__.o(definition, key) && !__webpack_require__.o(exports, key)) {
/******/ 					Object.defineProperty(exports, key, { enumerable: true, get: definition[key] });
/******/ 				}
/******/ 			}
/******/ 		};
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/hasOwnProperty shorthand */
/******/ 	(() => {
/******/ 		__webpack_require__.o = (obj, prop) => (Object.prototype.hasOwnProperty.call(obj, prop))
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/make namespace object */
/******/ 	(() => {
/******/ 		// define __esModule on exports
/******/ 		__webpack_require__.r = (exports) => {
/******/ 			if(typeof Symbol !== 'undefined' && Symbol.toStringTag) {
/******/ 				Object.defineProperty(exports, Symbol.toStringTag, { value: 'Module' });
/******/ 			}
/******/ 			Object.defineProperty(exports, '__esModule', { value: true });
/******/ 		};
/******/ 	})();
/******/ 	
/************************************************************************/
/******/ 	
/******/ 	// startup
/******/ 	// Load entry module and return exports
/******/ 	// This entry module can't be inlined because the eval devtool is used.
/******/ 	var __webpack_exports__ = __webpack_require__("./src/App.js");
/******/ 	
/******/ })()
;