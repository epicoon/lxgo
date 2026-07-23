// @lx:module lx.socket.WebSocketClient;

// @lx:namespace lx.socket;
class WebSocketClient {
    // @lx:const STATUS_NEW = 1;
    // @lx:const STATUS_IN_CONNECTING = 2;
    // @lx:const STATUS_CONNECTED = 3;
    // @lx:const STATUS_CLOSED = 4;
    // @lx:const STATUS_DISCONNECTED = 5;
    // @lx:const STATUS_ACCESS_DENIED = 6;
    // @lx:const STATUS_WAITING_FOR_RECONNECTING = 7;

    /**
     * @param config {Object: {
     *     port {Number},
     *     [url = ''] {String},
     *     [route = ''] {String},
     *     [protocol = 'ws'] {String},
     *     [reconnect = true] {Boolean},
     *     [buffer = true] {Boolean|Number} (: If Number - buffer life in milliseconds :),
     *     handlers {Object: {
     *         [onOpen] {Function|Tuple:[Object, Function]},
     *         [onConnected] {Function|Tuple:[Object, Function]},
     *         [onActionError] {Function|Tuple:[Object, Function]},
     *         [onAccessDenied] {Function|Tuple:[Object, Function]},
     *         [onClose] {Function|Tuple:[Object, Function]},
     *         [onError] {Function|Tuple:[Object, Function]},
     *         [onBeforeSend] {Function|Tuple:[Object, Function]},
     *         [onMessage] {Function|Tuple:[Object, Function]},
     *         [onChannelCreated] {Function|Tuple:[Object, Function]},
     *         [onChannelEntered] {Function|Tuple:[Object, Function]},
     *         [onChannelLeft] {Function|Tuple:[Object, Function]},
     *         [onChannelClosed] {Function|Tuple:[Object, Function]},
     *         [onChannelMessage] {Function|Tuple:[Object, Function]},
     *         [onChannelMateEntered] {Function|Tuple:[Object, Function]},
     *         [onChannelMateUpdated] {Function|Tuple:[Object, Function]},
     *         [onChannelMateLeft] {Function|Tuple:[Object, Function]},
     *         [onChannelMateDisconnected] {Function|Tuple:[Object, Function]},
     *         [onChannelMateReconnected] {Function|Tuple:[Object, Function]},
     *         [onChannelEvent] {lx.socket.EventListener|Function|Tuple:[Object, Function]}
     *     }}
     * }}
     */
    constructor(config) {
        this._protocol = config.protocol || 'ws';
        this._url = config.url || document.location.hostname;
        this._port = config.port || null;
        this._route = config.route || '';

        this._availableChannels = {};
        this._channels = new lx.socket.ChannelsList(this);

        this._socket = null;
        this._status = lx.self(STATUS_NEW);
        this._isReadyForClose = false;
        this._id = null;
        this._reconnect = lx.getFirstDefined(config.reconnect, true);
        this._reconnectionAllowed = false;
        this._errors = [];
        this._buffer = null;
        this._timer = null;
        _defineBuffer(this, config);

        this._onOpen = [];
        this._onConnected = [];
        this._onActionError = [];
        this._onAccessDenied = [];
        this._onClose = [];
        this._onError = [];
        this._onBeforeSend = [];
        this._onMessage = [];
        this._onChannelCreated = [];
        this._onChannelEntered = [];
        this._onChannelLeft = [];
        this._onChannelClosed = [];
        this._onChannelMessage = [];
        this._onChannelMateEntered = [];
        this._onChannelMateUpdated = [];
        this._onChannelMateDisconnected = [];
        this._onChannelMateReconnected = [];
        this._onChannelMateLeft = [];
        this._onChannelEvent = [];

        this._onChannel = {
            message: {},
            mateEntered: {},
            mateUpdated: {},
            mateDisconnected: {},
            mateReconnected: {},
            mateLeave: {},
            event: {},
            closed: {},
        };

        this.__qCounter = 0;
        this._qBuffer = {};

        if (config.handlers) {
            let handlers = config.handlers,
                methods = ['onOpen', 'onConnected', 'onActionError', 'onAccessDenied',
                    'onClose', 'onError', 'onBeforeSend', 'onMessage',
                    'onChannelCreated', 'onChannelEntered', 'onChannelLeft', 'onChannelClosed',
                    'onChannelMessage', 'onChannelMateEntered', 'onChannelMateUpdated',
                    'onChannelMateLeft', 'onChannelMateDisconnected', 'onChannelMateReconnected',
                    'onChannelEvent',
                ];
            for (let handlerName in handlers) {
                if (!methods.includes(handlerName)) continue;
                this[handlerName](handlers[handlerName]);
            }
        }
    }

    setPort(port) {
        this._port = port;
    }

    getUrl() {
        let path = this._protocol + '://' + this._url;
        return (this._port === null)
            ? path + (this._route ? '/' + this._route : '')
            : path +  ':' + this._port + (this._route ? '/' + this._route : '')
    }

    getId() {
        return this._id;
    }

    isConnected() {
        return this._status === lx.self(STATUS_CONNECTED);
    }

    hasErrors() {
        return this._errors.len;
    }

    getErrors() {
        return this._errors;
    }

    /**
     * @abstract
     * @returns {Object|null}
     */
    getAuthData() {
        return null;
    }

    /**
     * @abstract
     * @returns {Object|null}
     */
    getSharedData() {
        return null;
    }

    connect() {
        if (this.isConnected()) return;

        let url = this.getUrl();
        if (url === false) {
            let msg = 'Connection is unavailable. You have to define port and channel. Current port: ';
            msg += (this._port === null)
                ? 'undefined'
                : this._port;
            msg += '. Current channel: ';
            msg += '.';
            this._errors.push(msg);
            return;
        }

        if (window.MozWebSocket) this._socket = new MozWebSocket(url);
        else if (window.WebSocket) this._socket = new WebSocket(url);
        this._socket.binaryType = 'blob';
        this._status = lx.self(STATUS_IN_CONNECTING);

        _setSocketHandlers(this);
    }

    reconnect() {
        if (this._status !== lx.self(STATUS_CLOSED) && this._status !== lx.self(STATUS_DISCONNECTED)) return;
        this.connect();
    }
    
    close() {
        if (this._reconnectionAllowed && this._reconnect) {
            let map = lx.app.storage.get('lxsocket') || {},
                key = this.getUrl();            
            if (map.reconnect && (key in map.reconnect) && map.reconnect[key] == this._id) {
                delete map.reconnect[key];
                lx.app.storage.set('lxsocket', map);
            }
        }

        if (!this.isConnected()) return;
        _sendData(this, {__lxws_action__:'close'}, true);
    }

    break() {
        if (!this.isConnected()) return;
        _sendData(this, {__lxws_action__:'break'}, true);
    }

    request(route, params = {}) {
        let key = _getRequestKey(this),
            msg = {
            __lxws_request__: { route, key },
            __data__: params,
        };        
        let handler = new RequestHandler(this, key);
        _sendData(this, msg);
        return handler;
    }

    /**
     * @param {String} key
     * @returns {lx.socket.Channel|null}
     */
    getChannel(key) {
        return this._channels.get(key);
    }

    /**
     * Every channel this connection knows about - populated from the last
     * connect/reconnect, and live-updated as onChannelCreated fires
     * afterwards (own or someone else's public channel - see
     * onChannelCreated below). Includes channels you've created yourself
     * (public or private - you inherently know about your own), and every
     * public channel server-wide, whether you've entered it or not - but
     * not a "you're a member" list (see getChannel()/enterChannel() for
     * that; a channel you're actually in is both here *and* there). `data`
     * is each channel's shared data (e.g. a human-readable name/icon/type) -
     * enough to render a list of joinable channels without having to enter
     * them first.
     * @returns {Array<{key: String, data: Object}>}
     */
    getAvailableChannels() {
        return Object.entries(this._availableChannels).map(([key, data]) => ({ key, data }));
    }

    /**
     * Join a channel that already exists. The local lx.socket.Channel
     * is only created once the server acknowledges (see the 'enterChannel'
     * action handling in _setSocketHandlerOnMessage).
     * @param {String} channelKey
     * @param {Object} [sharedData]
     */
    enterChannel(channelKey, sharedData = null) {
        let data = { __lxws_action__: 'enterChannel', channelKey };
        if (sharedData !== null) data.sharedData = sharedData;
        _sendData(this, data, true);
    }

    /**
     * Create a new channel.
     * The creator is automatically entered into it on success,
     * same as a manual enterChannel(). initData isn't stored on the channel
     * or sent to anyone - it only exists for the server-side to inspect.
     * A proprietary channel closes (everyone still in it gets onChannelClosed)
     * the moment its creator leaves - explicitly via leaveChannel(), or by
     * disconnecting for good (not just a temporary drop still eligible to
     * reconnect). A non-proprietary channel has no owner in that sense - it
     * closes on its own once it's been empty for a while.
     * @param {Boolean} [public_=false]
     * @param {Boolean} [proprietary=false]
     * @param {Object} [sharedData]
     * @param {Object} [initData]
     */
    createChannel(public_ = false, proprietary = false, sharedData = null, initData = null) {
        let data = { __lxws_action__: 'createChannel', public: public_, proprietary };
        if (sharedData !== null) data.sharedData = sharedData;
        if (initData !== null) data.initData = initData;
        _sendData(this, data, true);
    }

    /**
     * Leave a channel previously joined via enterChannel() (or the default
     * one). The local lx.socket.Channel is only dropped once the server
     * acknowledges - not right after sending, so a failed leave on the
     * server side can't leave the client believing it left when it didn't.
     * @param {String} channelKey
     */
    leaveChannel(channelKey) {
        _sendData(this, { __lxws_action__: 'leaveChannel', channelKey }, true);
    }

    onPromisedConnection(callback) {
        if (this.isConnected()) {
            callback();
            return;
        }
        this.onConnected(callback);
    }

    onOpen(callback) { this._onOpen.push(callback) }
    onConnected(callback) { this._onConnected.push(callback) }

    /**
     * Fires for any __lxws_action__ the server rejected (unknown action,
     * malformed field, unknown channel for enterChannel/leaveChannel, ...) -
     * event.getData() is {action: String, error: String}.
     */
    onActionError(callback) { this._onActionError.push(callback) }

    onAccessDenied(callback) { this._onAccessDenied.push(callback) }
    onClose(callback) { this._onClose.push(callback) }
    onError(callback) { this._onError.push(callback) }
    onBeforeSend(callback) { this._onBeforeSend.push(callback) }
    onMessage(callback) { this._onMessage.push(callback) }

    /**
     * Fires whenever a new channel becomes known to this connection - either
     * your own createChannel() was just acknowledged (key is the
     * server-generated key), or a *different* connection created a new
     * public channel. Either way, event.getData() is {key: String,
     * data: Object} (data is the channel's shared data) - already reflected
     * in getAvailableChannels() by the time this fires. If it was your own
     * createChannel(), you're already a member (socket.getChannel(key) is
     * usable); otherwise you've merely learned it exists - enterChannel(key)
     * is still needed to join.
     */
    onChannelCreated(callback) { this._onChannelCreated.push(callback) }

    /**
     * Fires once the server acknowledges a successful enterChannel() -
     * event.getData() is {channel: lx.socket.Channel}.
     */
    onChannelEntered(callback) { this._onChannelEntered.push(callback) }

    /**
     * Fires once the server acknowledges a leaveChannel() (including when
     * it was a no-op because the connection wasn't a member) -
     * event.getData() is {channelKey: String}.
     */
    onChannelLeft(callback) { this._onChannelLeft.push(callback) }

    /**
     * Fires when a channel you were in gets closed server-side, including a
     * proprietary channel whose creator left for good (see createChannel),
     * or a non-proprietary one auto-closed after sitting empty - the latter
     * you'd only see if you're in it right as it happens, since being empty
     * is exactly what triggers it. event.getData() is {channelKey: String,
     * code: String} - "200" for the creator-left case, "100" for everything
     * else this package triggers itself (the empty-channel sweep, or an
     * explicit Close() call with no more specific code), or any other string
     * an application chose for its own close call - the channel is
     * already gone from getChannel()/getAvailableChannels() by the time this
     * fires. Accepts an optional leading channelKey to scope to one channel,
     * like onChannelMessage.
     * @param {String|Function|Tuple} channelKey omit to receive from every channel
     * @param {Function|Tuple} [callback]
     */
    onChannelClosed(channelKey, callback) {
        if (lx.isFunction(channelKey)) this._onChannelClosed.push(channelKey);
        else {
            if (!(channelKey in this._onChannel.closed))
                this._onChannel.closed[channelKey] = [];
            this._onChannel.closed[channelKey].push(callback);
        }
    }

    onChannelMessage(channelKey, callback) {
        if (lx.isFunction(channelKey)) this._onChannelMessage.push(channelKey);
        else {
            if (!(channelKey in this._onChannel.message))
                this._onChannel.message[channelKey] = [];
            this._onChannel.message[channelKey].push(callback);
        }
    }
    onChannelMateEntered(channelKey, callback) {
        if (lx.isFunction(channelKey)) this._onChannelMateEntered.push(channelKey);
        else {
            if (!(channelKey in this._onChannel.mateEntered))
                this._onChannel.mateEntered[channelKey] = [];
            this._onChannel.mateEntered[channelKey].push(callback);
        }
    }
    onChannelMateUpdated(channelKey, callback) {
        if (lx.isFunction(channelKey)) this._onChannelMateUpdated.push(channelKey);
        else {
            if (!(channelKey in this._onChannel.mateUpdated))
                this._onChannel.mateUpdated[channelKey] = [];
            this._onChannel.mateUpdated[channelKey].push(callback);
        }
    }
    onChannelMateLeft(channelKey, callback) {
        if (lx.isFunction(channelKey)) this._onChannelMateLeft.push(channelKey);
        else {
            if (!(channelKey in this._onChannel.mateLeave))
                this._onChannel.mateLeave[channelKey] = [];
            this._onChannel.mateLeave[channelKey].push(callback);
        }
    }
    onChannelMateDisconnected(channelKey, callback) {
        if (lx.isFunction(channelKey)) this._onChannelMateDisconnected.push(channelKey);
        else {
            if (!(channelKey in this._onChannel.mateDisconnected))
                this._onChannel.mateDisconnected[channelKey] = [];
            this._onChannel.mateDisconnected[channelKey].push(callback);
        }
    }
    onChannelMateReconnected(channelKey, callback) {
        if (lx.isFunction(channelKey)) this._onChannelMateReconnected.push(channelKey);
        else {
            if (!(channelKey in this._onChannel.mateReconnected))
                this._onChannel.mateReconnected[channelKey] = [];
            this._onChannel.mateReconnected[channelKey].push(callback);
        }
    }

    /**
     * Handles a lx.socket.Channel.trigger()'d event - callback can be a
     * plain function, a [thisObj, function] tuple, or an
     * lx.socket.EventListener instance (dispatched by event name via its
     * own .processEvent(), see the EventListener class below).
     * @param {String|Function|lx.socket.EventListener} channelKey omit to
     *   receive events from every channel
     * @param {Function|lx.socket.EventListener} [callback]
     */
    onChannelEvent(channelKey, callback) {
        if (lx.isFunction(channelKey) || channelKey instanceof lx.socket.EventListener) {
            callback = channelKey;
            if (callback instanceof lx.socket.EventListener) callback.setSocket(this);
            this._onChannelEvent.push(callback);
        } else {
            if (callback instanceof lx.socket.EventListener) callback.setSocket(this);
            if (!(channelKey in this._onChannel.event))
                this._onChannel.event[channelKey] = [];
            this._onChannel.event[channelKey].push(callback);
        }
    }

    static dropReconnectionsData(list=null) {
        if (list === null) {
            lx.app.storage.remove('lxsocket')
            return;
        }
        let lxSocketData = lx.app.storage.get('lxsocket') || {};
        for (let i in list) {
            let url = list[i];
            if (url in lxSocketData.reconnect)
                delete lxSocketData.reconnect[url];
        }
        lx.app.storage.set('lxsocket', lxSocketData);
    }

    static filterReconnectionsData(list) {
        let lxSocketData = lx.app.storage.get('lxsocket') || {},
            newReconnect = {};
        for (let i in list) {
            let url = list[i];
            if (url in lxSocketData.reconnect)
                newReconnect[url] = lxSocketData.reconnect[url];
        }
        lxSocketData.reconnect = newReconnect;
        lx.app.storage.set('lxsocket', lxSocketData);
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ConnectionEvent
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class ConnectionEvent {
    constructor(eventName, socket, payload = {}) {
        this._name = eventName;
        this._socket = socket;
        this._data = payload;
    }

    getName() {
        return this._name;
    }

    getData() {
        return this._data;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Message
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class Message
{
    constructor(socket, conf) {
        this._socket = socket;
        this._data = conf.__lxws_message__ || null;
    }

    getData() {
        return this._data;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ChannelsList
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class ChannelsList
{
    /**
     * @param {WebSocketClient} socket 
     */
    constructor(socket) {
        this.socket = socket;
        this.list = {};
    }

    /**
     * @param {String} key
     * @param {Object} data
     * @param {Object} matesData
     * @returns {Channel}
     */
    create(key, data, matesData) {
        if (this.has(key)) {
            console.error('Channel "' + key + '" already exists');
            return;
        }
        const ch = new lx.socket.Channel(this.socket, key, data, matesData);
        this.add(ch);
        return ch;
    }

    reset() {
        this.list = {};
    }

    /**
     * @param {Channel} channel
     */
    add(channel) {
        this.list[channel.getKey()] = channel;
    }

    /**
     * @param {String} key
     * @returns {Boolean}
     */
    has(key) {
        return key in this.list;
    }

    /**
     * @param {String} key
     * @returns {Channel|null}
     */
    get(key) {
        return this.list[key] || null;
    }

    /**
     * @param {String} key
     */
    remove(key) {
        delete this.list[key];
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Channel
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class Channel
{
    /**
     * @param {WebSocketClient} socket
     * @param {String} key
     * @param {Object} data
     * @param {Array} matesData
     * @returns {Channel}
     */
    constructor(socket, key, data, matesData) {
        this.socket = socket;
        this.key = key;
        this.data = data;
        this.mates = {};
        this.disconnected = {};
        matesData.forEach(d=>this.registerMate(d.id, d.data));
    }

    /**
     * @returns {String}
     */
    getKey() {
        return this.key;
    }

    /**
     * @returns {Object}
     */
    getSharedData() {
        return this.data;
    }

    /**
     * @returns {Object <String:ChannelMate>}
     */
    getMates() {
        return this.mates;
    }

    /**
     * @returns {ChannelMate|null}
     */
    getMate(id) {
       return this.mates[id] || null;       
    }

    /**
     * @returns {ChannelMate}
     */
    getLocalMate() {
        return this.mates[this.socket._id];
    }

    /**
     * @param {String} mateId
     * @param {Object} mateData
     * @returns {ChannelMate}
     */
    registerMate(mateId, mateData) {
        this.mates[mateId] = new lx.socket.ChannelMate(this.socket, mateId, mateData);
        return this.mates[mateId];
    }

    /**
     * @param {String} mateId 
     * @returns {ChannelMate|null}
     */
    disconnectMate(mateId) {
        if (!(mateId in this.mates)) return null;
        this.disconnected[mateId] = this.mates[mateId];
        delete this.mates[mateId];
        return this.disconnected[mateId];
    }

    /**
     * @param {String} mateId 
     * @returns {ChannelMate|null}
     */
    reconnectMate(mateId) {
        if (!(mateId in this.disconnected)) return null;
        this.mates[mateId] = this.disconnected[mateId];
        delete this.disconnected[mateId];
        return this.mates[mateId];
    }

    /**
     * @param {String} mateId 
     * @returns {ChannelMate|null}
     */
    dropMate(mateId) {
        let m = null;
        if (mateId in this.mates) {
            m = this.mates[mateId];
            delete this.mates[mateId];
        }
        if (mateId in this.disconnected) {
            m = this.disconnected[mateId];
            delete this.disconnected[mateId];
        }
        return m;
    }

    updateSharedData(data) {
        _sendData(this.socket, _prepareMsg(this, 'sharedData', data));
    }

    send(data, receivers = null, returnToSender = false, privateMode = false) {
        let msg = _prepareMsg(this, 'message', data);
        _applyDelivery(msg, receivers, returnToSender, privateMode);
        _sendData(this.socket, msg);
    }

    /**
     * @param {String} eventName
     * @param {Object} [data]
     * @param {Array} [receivers] same shape as send() - omit to reach the whole channel
     * @param {Boolean} [returnToSender=true]
     * @param {Boolean} [privateMode=false]
     */
    trigger(eventName, data = {}, receivers = null, returnToSender = true, privateMode = false) {
        let msg = _prepareMsg(this, 'event', data);
        msg.__metaData__.event = eventName;
        _applyDelivery(msg, receivers, returnToSender, privateMode);
        _sendData(this.socket, msg);
    }
}

/**
 * @param {Channel} channel
 * @param {String} type
 * @param {Object} data
 * @returns {Object}
 */
function _prepareMsg(channel, type, data) {
    return {
        __lxws_channel__: type,
        __data__: data,
        __metaData__:{
            channel: channel.getKey(),
        }
    };
}

/**
 * Shared by Channel.send()/trigger() - was previously duplicated inline in
 * send() only, so trigger()'s receivers/returnToSender/privateMode
 * parameters were silently ignored (a real bug, not just duplication).
 * @param {Object} msg
 * @param {Array|null} receivers
 * @param {Boolean} returnToSender
 * @param {Boolean} privateMode
 */
function _applyDelivery(msg, receivers, returnToSender, privateMode) {
    if (receivers) {
        let receiverIds = [];
        receivers.forEach(r=>{
            if (r instanceof lx.socket.ChannelMate) {
                receiverIds.push(r.getId());
            } else if (lx.isString(r)) {
                receiverIds.push(r);
            }
        });
        msg.__metaData__.receivers = receiverIds;
    }
    msg.__metaData__.returnToSender = returnToSender;
    msg.__metaData__.private = (!msg.__metaData__.receivers) ? false : privateMode;
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ChannelMate
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class ChannelMate
{
    constructor(socket, id, data) {
        this._socket = socket;
        this._id = id;
        this._isLocal = this._id == this._socket.getId();
        this._params = {};

        this._params = data.lxClone();
        if (this._params._id)
            delete this._params._id;
        if (this._params._isLocal)
            delete this._params._isLocal;
        for (let key in data) {
            Object.defineProperty(this, key, {
                get: function() {
                    return this._params[key];
                }
            });
        }
    }

    getId() {
        return this._id;
    }

    getSocket() {
        return this._socket;
    }

    isLocal() {
        return this._isLocal;
    }

    setSharedParam(key, value) {
        if (key == '_id' || key == '_isLocal')
            return;
        if (!(key in this._params)) {
            Object.defineProperty(this, key, {
                get: function() {
                    return this._params[key];
                }
            });
        }
        this._params[key] = value;
    }

    setSharedData(data) {
        for (let i in data)
            this.setSharedParam(i, data[i]);
    }

    hasParam(name) {
        return (name in this._params);
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ChannelStdEvent
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class ChannelStdEvent {
    constructor(eventName, socket, channel, mate) {
        this._name = eventName;
        this._socket = socket;
        this._channel = channel;
        this._mate = mate;
    }

    getName() {
        return this._name;
    }

    getChannel() {
        return this._channel;
    }

    getMate() {
        return this._mate;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ChannelMessage
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class ChannelMessage extends lx.socket.Message {
    constructor(channel, conf) {
        super(channel.socket, conf);
        this._data = conf.data;

        this.channel = channel;
        this._fromId = conf.from;
        this._addressed = conf.addressed;
        this._private = conf.private;
    }

    getData() {
        return this._data;
    }

    getAuthor() {
        return this.channel.getMate(this._fromId);
    }

    isPrivate() {
        return this._private;
    }

    isAddressed() {
        return this._addressed;
    }

    isFromMe() {
        return this.getAuthor().isLocal();
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * EventListener
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class EventListener
{
    constructor() {
        this._socket = null;
    }

    setSocket(socket) {
        this._socket = socket;
    }

    processEvent(event) {
        let name = event.getName();
        let methodName = 'on-' + name;
        methodName = methodName.replace(/[-_](.)/g, function(str, match) {
            return match.toUpperCase();
        });

        event = this.preprocessEvent(event);

        if (this.lxHasMethod(methodName)) this[methodName](event);
        else this.onEvent(event);
    }

    preprocessEvent(event) {
        return event;
    }

    onEvent(event) {
        // pass
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ChannelEvent
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class ChannelEvent extends lx.socket.ChannelMessage
{
    constructor(channel, conf) {
        super(channel, conf);

        this._name = conf.event;
    }

    getName() {
        return this._name;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

class RequestHandler {
    constructor(socket, key) {
        this.socket = socket;
        this.key = key;
    }
    
    then(callback) {
        this.socket._qBuffer[this.key] = callback;
    }
}

function _defineBuffer(self, config) {
    let buffer = lx.getFirstDefined(config.buffer, true);
    if (buffer === false) return;

    if (buffer === true) buffer = 200;
    self._buffer = [];
    self._timer = new lx.Timer({
        period: buffer,
        countPeriods: false
    });
    self._timer._socket = self;
    self._timer.onCycleEnd(function() {
        if (!this._socket._buffer.len) return;

        let buffer = this._socket._buffer;
        this._socket._buffer = [];

        if (buffer.len == 1)
            this._socket._socket.send(JSON.stringify(buffer[0]));
        else
            this._socket._socket.send(JSON.stringify({
                __multi__: true,
                __list__: buffer
            }));
    });
}

function _setSocketHandlers(self) {
    _setSocketHandlerOnOpen(self);
    _setSocketHandlerOnMessage(self);
    _setSocketHandlerOnClose(self);
    _setSocketHandlerOnError(self);
}

function _setSocketHandlerOnOpen(self) {
    if (self._socket === null) return;
    self._socket.onopen = function() {
        self._status = lx.socket.WebSocketClient.STATUS_CONNECTED;
        if (self._timer) {
            self._buffer = [];
            self._timer.start();
        }
        _runHandlers(self, 'open', self._onOpen);
    }
}

// Absorbs one {key, data[, connections]} channel entry - shared by
// connect/reconnect's "channels" list and createChannel's ack/announcement,
// all of which speak the same shape. "connections" present means this
// connection is a member (establish the local lx.socket.Channel); its
// absence just means the channel is available/known, not joined - either
// way it's recorded in _availableChannels (see getAvailableChannels()).
function _absorbChannelEntry(self, ch) {
    if (ch.connections) {
        let channel = self._channels.get(ch.key);
        if (!channel) self._channels.create(ch.key, ch.data, ch.connections);
    }
    self._availableChannels[ch.key] = ch.data;
}

function _setSocketHandlerOnMessage(self) {
    if (self._socket === null) return;
    self._socket.onmessage =(e)=>{
        let msg = JSON.parse(e.data);

        if (self._id === null) {
            _onHandshake(self, msg);
            return;
        }

        if (msg.__lxws_message__) {
            _runHandlers(self, new lx.socket.Message(self, msg), self._onMessage);
            return;
        }

        if (msg.__lxws_response__) {
            if (msg.key in self._qBuffer) {
                let f = self._qBuffer[msg.key];
                delete self._qBuffer[msg.key];
                let body = JSON.parse(msg.body);
                f({code:msg.code,headers:msg.headers,body});
            }
            return;
        }

        if (msg.__lxws_channel__) {
            let channel, mate;
            switch (msg.__lxws_channel__) {
                case 'mateEntered':
                    channel = self.getChannel(msg.channel);
                    if (!channel) return;
                    mate = channel.registerMate(msg.id, msg.data);
                    if (!mate) return;
                    _runChannelHandlers(self, channel, mate, 'mateEntered');
                    break;

                case 'mateUpdated':
                    channel = self.getChannel(msg.channel);
                    if (!channel) return;
                    mate = channel.getMate(msg.id);
                    if (!mate) return;
                    mate.setSharedData(msg.data);
                    _runChannelHandlers(self, channel, mate, 'mateUpdated');
                    break;

                case 'mateLeft':
                    channel = self.getChannel(msg.channel);
                    if (!channel) return;
                    mate = channel.dropMate(msg.id);
                    if (!mate) return;
                    _runChannelHandlers(self, channel, mate, 'mateLeave');
                    break;

                case 'mateDisconnected':
                    channel = self.getChannel(msg.channel);
                    if (!channel) return;
                    mate = channel.disconnectMate(msg.id);
                    if (!mate) return;
                    _runChannelHandlers(self, channel, mate, 'mateDisconnected');
                    break;

                case 'mateReconnected':
                    channel = self.getChannel(msg.channel);
                    if (!channel) return;
                    mate = channel.reconnectMate(msg.id);
                    if (!mate) return;
                    _runChannelHandlers(self, channel, mate, 'mateReconnected');
                    break;

                case 'closed': {
                    if (!self.getChannel(msg.channel)) return;
                    const event = new lx.socket.ConnectionEvent(
                        'channelClosed', self, { channelKey: msg.channel, code: msg.code }
                    );
                    self._channels.remove(msg.channel);
                    delete self._availableChannels[msg.channel];
                    _runHandlers(self, event, self._onChannelClosed);
                    if (msg.channel in self._onChannel.closed)
                        _runHandlers(self, event, self._onChannel.closed[msg.channel]);
                    break;
                }

                case 'message': {
                    const channel = self.getChannel(msg.channel);
                    if (!channel) return;
                    const msgObj = new lx.socket.ChannelMessage(channel, msg);
                    _runHandlers(self, msgObj, self._onChannelMessage);
                    if (channel.getKey() in self._onChannel.message)
                        _runHandlers(self, msgObj, self._onChannel.message[channel.getKey()]);
                    break;
                }

                case 'event': {
                    const eventChannel = self.getChannel(msg.channel);
                    if (!eventChannel) return;
                    const event = new lx.socket.ChannelEvent(eventChannel, msg);
                    _runEventHandlers(self, self._onChannelEvent, event);
                    if (msg.channel in self._onChannel.event)
                        _runEventHandlers(self, self._onChannel.event[msg.channel], event);
                    break;
                }
            }
            return;
        }

        if (msg.__lxws_action__) {
            if (msg.error) {
                _runHandlers(self, 'actionError', self._onActionError, { action: msg.__lxws_action__, error: msg.error });
                return;
            }
            switch (msg.__lxws_action__) {
                case 'connect':
                    if (msg.channels)
                        msg.channels.forEach(ch=>_absorbChannelEntry(self, ch));
                    _runHandlers(self, 'connected', self._onConnected);
                    break;

                case 'reconnect':
                    self._id = msg.idRestored;
                    _updateReconnectId(self);
                    if (msg.channels)
                        msg.channels.forEach(ch=>_absorbChannelEntry(self, ch));
                    _runHandlers(self, 'connected', self._onConnected);
                    break;

                case 'createChannel':
                    if (msg.channel) {
                        // Sent both to the creator (with "connections" - this
                        // connection is already a member) and to everyone else if the
                        // channel is public (no "connections" - just discoverable).
                        _absorbChannelEntry(self, msg.channel);
                        _runHandlers(self, 'channelCreated', self._onChannelCreated, { key: msg.channel.key, data: msg.channel.data });
                    }
                    break;

                case 'enterChannel':
                    if (msg.channel) {
                        const ch = msg.channel;
                        let channel = self._channels.get(ch.key);
                        if (!channel) channel = self._channels.create(ch.key, ch.data, ch.connections);
                        _runHandlers(self, 'channelEntered', self._onChannelEntered, { channel });
                    }
                    break;

                case 'leaveChannel':
                    if (msg.channelKey) {
                        self._channels.remove(msg.channelKey);
                        _runHandlers(self, 'channelLeft', self._onChannelLeft, { channelKey: msg.channelKey });
                    }
                    break;

                case 'close':
                    self._isReadyForClose = true;
                    self._socket.close();
                    break;

                case 'break':
                    self._socket.close();
                    break;
            }
            return;
        }
    };
}

function _updateReconnectId(self) {
    let key = self.getUrl(),
        lxSocketData = lx.app.storage.get('lxsocket') || {};
    if (!lxSocketData.reconnect) lxSocketData.reconnect = {};
    let oldConnectionId = lxSocketData.reconnect[key] || null;
    lxSocketData.reconnect[key] = self._id;
    lx.app.storage.set('lxsocket', lxSocketData);
    return oldConnectionId;
}

function _onHandshake(self, msg) {
    self._id = msg.id;

    if (msg.reconnectionAllowed) {
        self._reconnectionAllowed = true;
        self._reconnectionStep = 0;
        self._reconnectionNextStep = 1;

        let oldConnectionId = _updateReconnectId(self);
        if (oldConnectionId) {
            _sendReconnectionData(self, oldConnectionId);
            return;
        }
    }

    _sendConnectionData(self);
}

function _runHandlers(self, eventName, handlers, payload = {}) {
    if (!handlers.len) return;
    for (let i in handlers) {
        let handler = handlers[i],
            event = lx.isString(eventName)
                ? new lx.socket.ConnectionEvent(eventName, self, payload)
                : eventName;
        if (lx.isArray(handler))
            handler[1].call(handler[0], event);
        else handler.call(self, event);
    }
}

function _runChannelHandlers(self, channel, mate, eventName) {
    const event = new lx.socket.ChannelStdEvent(eventName, self, channel, mate)

    const commonHandlersKey = '_onChannel' + eventName.charAt(0).toUpperCase() + eventName.slice(1);
    if (commonHandlersKey in self)
        _runHandlers(self, event, self[commonHandlersKey]);
    else console.error('Unknown channel handlers:', commonHandlersKey);

    if (channel.getKey() in self._onChannel[eventName])
        _runHandlers(self, event, self._onChannel[eventName][channel.getKey()]);
}

// Unlike _runHandlers, a channel-event handler may also be an
// lx.socket.EventListener instance (see onChannelEvent) - it gets
// event-name-convention dispatch via .processEvent(), instead of being
// called directly.
function _runEventHandlers(self, handlers, event) {
    if (!handlers.len) return;
    for (let i in handlers) {
        let handler = handlers[i];
        if (handler instanceof lx.socket.EventListener) handler.processEvent(event);
        else if (lx.isArray(handler)) handler[1].call(handler[0], event);
        else handler.call(self, event);
    }
}

function _setSocketHandlerOnClose(self) {
    if (self._socket === null) return;
    self._socket.onclose =(e)=>{
        if (self._timer) {
            self._timer.stop();
            self._buffer = [];
        }

        if (e.code == 1002) _runHandlers(self, e, self._onAccessDenied);
        else _runHandlers(self, e, self._onClose);
        self._socket = null;
        self._id = null;
        self._availableChannels = {};
        self._channels.reset();

        if (e.code == 1002) {
            self._status = lx.socket.WebSocketClient.STATUS_CLOSED;
            self._isReadyForClose = false;
        } else if (self._isReadyForClose) {
            self._status = lx.socket.WebSocketClient.STATUS_ACCESS_DENIED;
            self._isReadyForClose = false;
        } else {
            self._status = lx.socket.WebSocketClient.STATUS_DISCONNECTED;
            _afterDisconnect(self);
        }
    };
}

function _setSocketHandlerOnError(self) {
    if (self._socket === null) return;
    self._socket.onerror =(e)=>{
        if (self._timer) {
            self._timer.stop();
            self._buffer = [];
        }

        self._errors.push(e);
        _runHandlers(self, e, self._onError);
        self._socket = null;
        self._id = null;
        self._availableChannels = {};
        self._channels.reset();
        self._status = lx.socket.WebSocketClient.STATUS_DISCONNECTED;
        _afterDisconnect(self);
    };
}

function _afterDisconnect(self) {
    if (!self._reconnectionAllowed || !self._reconnect) return;

    let duration = self._reconnectionStep,
        next = self._reconnectionStep + self._reconnectionNextStep;
    self._reconnectionStep = self._reconnectionNextStep;
    self._reconnectionNextStep = next;

    if (duration) {
        self._status = lx.socket.WebSocketClient.STATUS_WAITING_FOR_RECONNECTING;
        let timer = new lx.Timer(duration * 500);
        timer.onCycleEnd(()=>{
            self._status = lx.socket.WebSocketClient.STATUS_DISCONNECTED;
            self.reconnect();
            timer.stop();
            delete self.timer;
        });
        timer.start();
        self.timer = timer;
    } else self.reconnect();
}

/**
 * @param {WebSocketClient} self
 * @returns {String}
 */
function _getRequestKey(self) {
    if (self.__qCounter == 999999) self.__qCounter = 0;
    let result = self.__qCounter;
    self.__qCounter++;
    result += '_' + lx.Math.randomInteger(0, 999999) + '_' + Date.now() + '_' + (new Date).getMilliseconds();
    return result;
}

function _getConnectionData(self) {
    let data = { __lxws_action__: 'connect' };
    let ad = self.getAuthData();
    if (ad !== null) data.auth = ad;
    let sd = self.getSharedData();
    if (sd !== null) data.shared = sd;
    return data;
}

function _sendConnectionData(self) {
    _sendData(self, _getConnectionData(self), true);
}

function _sendReconnectionData(self, oldId) {
    let data = _getConnectionData(self);
    data.__lxws_action__ = 'reconnect';
    data.oldConnectionId = oldId;
    _sendData(self, data, true);
}

function _sendData(self, data, force = false) {
    if (!self.isConnected()) return;

    if (!('__lxws_action__' in data)) {
        for (let i in self._onBeforeSend) {
            let handler = self._onBeforeSend[i],
                event = new lx.socket.ConnectionEvent('beforeSend', self, data);
            if (handler(event) === false) return;
        }
    }

    if (!force && self._timer) self._buffer.push(data);
    else self._socket.send(JSON.stringify(data));
}
