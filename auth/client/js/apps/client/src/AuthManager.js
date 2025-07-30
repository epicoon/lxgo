export default class AuthManager {
    constructor(app) {
        this.app = app;
        this.settings = {};
        this.accessToken = null;
        this.refreshToken = null;
        this.active = _loadSettings(this);
        this.redirecting = false;
    }

    /**
     * @returns {Token}
     */
    getAccessToken() {
        return _getAccessToken(this);
    }

    /**
     * @returns {Token}
     */
    getRefreshToken() {
        return _getRefreshToken(this);
    }

    /**
     * @trigger lxAuth.TOKENS_FOUND
     * @trigger lxAuth.TOKENS_NOT_FOUND
     */
    checkTokens() {
        if (!this.active) return;

        const accessToken = _getAccessToken(this);
        if (!accessToken.isActive()) {
            const refreshToken = _getRefreshToken(this);
            if (!refreshToken.isActive()) {
                this.app.trigger('TokensNotFound');
                return;    
            }
        }

        this.app.trigger('TokensFound');
    }

    /**
     * @returns {bool}
     */
    async refreshTokens() {
        const refreshToken = _getRefreshToken(this);
        if (!refreshToken.isActive()) {
            console.error("Refresh auth token invalid");
            return false;
        }

        const response = await fetch(this.settings.refresh_path, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                refresh_token: refreshToken.value
            })
        });

        if (!response.ok) {
            console.error("Refresh auth tokens failed");
            return false;
        }

        const data = await response.json();
        const accessToken = _getAccessToken(this);
        accessToken.init(data.access_token, data.access_token_expired);
        accessToken.toStorage('lxAuthAccessToken');
        refreshToken.init(data.refresh_token, data.refresh_token_expired);
        refreshToken.toStorage('lxAuthRefreshToken');

        return true;
    }

    async goToAuth() {
        if (this.redirecting) return;
        this.redirecting = true;

        const state = await _genState(this);
        if (state === null) {
            console.error("Can not redirect");
            return;
        }

        const authData = this.settings;
        _postRedirect(`${authData.server}/auth`, {
            response_type: 'code',
            client_id: authData.id,
            redirect_uri: authData.redirect_uri,
            state
        });
    }

    /**
     * @trigger lxAuth.TOKENS_REMOVED
     */
    async logOut() {
        const refreshToken = _getRefreshToken(this);
        if (!refreshToken.isActive()) {
            _dropTokens(this);
            this.app.trigger('TokensRemoved');
            return;
        }

        const accessToken = _getAccessToken(this);
        const response = await fetch(this.settings.logout_path, {
            method: 'GET',
            headers: { 'Authorization': 'Bearer ' + accessToken.value }
        });

        if (response.ok) {
            _dropTokens(this);
            this.app.trigger('TokensRemoved');
        }
    }

    /**
     * @returns {Object: {
     *     {bool} success,
     *     {string} login,
     *     {Object} data
     * }}
     */
    async getUserData() {
        const response = await this.app.fetch(this.settings.user_data_path);
        if (!response || !response.ok) {
            console.error('Fetching user data failed');
            return {success: false};
        }
        const data = await response.json();
        data.success = true;
        return data;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/**
 * @private
 * @param {AuthManager} self 
 * @returns {bool}
 */
function _loadSettings(self) {
    const as = window._lxauth_settings;
    if (!as || as === '') {
        console.error('Auth settings are not available');
        return false;
    }

    delete window._lxauth_settings;

    let sett;
    try {
        sett = JSON.parse(as);
    } catch (e) {
        console.error('Can not parse auth settings');
        return false;
    }

    self.settings = sett;
    return true;
}

/**
 * @param {AuthManager} self 
 * @returns {string}
 */
async function _genState(self) {
    const response = await fetch(self.settings.state_path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uri: window.location.href })
    });
    if (!response || !response.ok) {
        console.error('Fetching state failed');
        return null;
    }
    const data = await response.json();
    return data.state;
}

/**
 * @private
 * @param {AuthManager} self 
 * @returns {Token}
 */
function _getAccessToken(self) {
    if (self.accessToken === null) {
        _readToken(self, 'accessToken', 'lxAuthAccessToken');
    }
    return self.accessToken;
}

/**
 * @private
 * @param {AuthManager} self 
 * @returns {Token}
 */
function _getRefreshToken(self) {
    if (self.refreshToken === null) {
        _readToken(self, 'refreshToken', 'lxAuthRefreshToken');
    }
    return self.refreshToken;
}

/**
 * @private
 * @param {AuthManager} self
 * @param {string} selfKey
 * @param {string} lsKey
 */
function _readToken(self, selfKey, lsKey) {
    if (!self.active) return false;

    self[selfKey] = new Token();

    let tokenData = localStorage.getItem(lsKey);
    if (!tokenData) {
        return;
    }

    try {
        tokenData = JSON.parse(tokenData)
    } catch (e) {
        console.error(e);
        return;
    }

    self[selfKey].init(tokenData[0], tokenData[1]);
}

/**
 * @private
 * @param {AuthManager} self 
 */
function _dropTokens(self) {
    localStorage.removeItem('lxAuthAccessToken');
    localStorage.removeItem('lxAuthRefreshToken');
    self.accessToken = null;
    self.refreshToken = null;
}

/**
 * @private
 * @param {string} url
 * @param {Object} params
 */
function _postRedirect(url, params) {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = url;

    for (const key in params) {
        if (params.hasOwnProperty(key)) {
            const input = document.createElement("input");
            input.type = "hidden";
            input.name = key;
            input.value = params[key];
            form.appendChild(input);
        }
    }

    document.body.appendChild(form);
    form.submit();
}

class Token {
    constructor() {
        this.exists = false;
        this.value = null;
        this.expiresAt = null;
    }

    /**
     * @param {string} value 
     * @param {number} expiresAt 
     */
    init(value, expiresAt) {
        this.value = value;
        this.expiresAt = +expiresAt;
        this.exists = true;
    }

    /**
     * @param {string} key 
     */
    toStorage(key) {
		localStorage.setItem(key, '["'+this.value+'", '+this.expiresAt+']');
    }

    /**
     * @returns {bool}
     */
    isExpired() {
        const currentTime = Math.floor(Date.now() / 1000);
        return this.expiresAt <= currentTime;
    }

    /**
     * @returns {bool}
     */
    isActive() {
        return this.exists && !this.isExpired();
    }
}
