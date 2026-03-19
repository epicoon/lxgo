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
     *         [onAccessDenied] {Function|Tuple:[Object, Function]},
     *         [onClose] {Function|Tuple:[Object, Function]},
     *         [onError] {Function|Tuple:[Object, Function]},
     *         [onBeforeSend] {Function|Tuple:[Object, Function]},
     *         [onMessage] {Function|Tuple:[Object, Function]},
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
        this._onAccessDenied = [];
        this._onClose = [];
        this._onError = [];
        this._onBeforeSend = [];
        this._onMessage = [];
        this._onChannelMessage = [];
        this._onChannelMateEntered = [];
        this._onChannelMateUpdated = [];
        this._onChannelMateDisconnected = [];
        this._onChannelMateReconnected = [];
        this._onChannelMateLeft = [];

        this._onChannel = {
            message: {},
            mateEntered: {},
            mateUpdated: {},
            mateDisconnected: {},
            mateReconnected: {},
            mateLeave: {},
        };

        //TODO events are not implemented. Do we need common events? Maybe only channel events
        //this._onEvent = [];

        this.__qCounter = 0;
        this._qBuffer = {};

        if (config.handlers) {
            let handlers = config.handlers,
                methods = ['onOpen', 'onConnected', 'onAccessDenied',
                    'onClose', 'onError', 'onBeforeSend', 'onMessage',
                    'onChannelMessage', 'onChannelMateEntered', 'onChannelMateUpdated',
                    'onChannelMateLeft', 'onChannelMateDisconnected', 'onChannelMateReconnected',

                    //TODO events are not implemented. Do we need common events? Maybe only channel events
                    //'onChannelEvent', 
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

    onPromisedConnection(callback) {
        if (this.isConnected()) {
            callback();
            return;
        }
        this.onConnected(callback);
    }

    onOpen(callback) { this._onOpen.push(callback) }
    onConnected(callback) { this._onConnected.push(callback) }
    onAccessDenied(callback) { this._onAccessDenied.push(callback) }
    onClose(callback) { this._onClose.push(callback) }
    onError(callback) { this._onError.push(callback) }
    onBeforeSend(callback) { this._onBeforeSend.push(callback) }
    onMessage(callback) { this._onMessage.push(callback) }

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

    //TODO events are not implemented. Do we need common events? Maybe only channel events
    // onChannelEvent(callback) {
    //     if (callback instanceof lx.socket.EventListener)
    //         callback.setSocket(this);
    //     this._onEvent.push(callback);
    // }

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
        this.payload = payload;
    }

    getName() {
        return this._name;
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
        matesData.forEach(d=>this.register(d.id, d.data));
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
    register(mateId, mateData) {
        this.mates[mateId] = new lx.socket.ChannelMate(this.socket, mateId, mateData);
        return this.mates[mateId];
    }

    /**
     * @param {String} mateId 
     * @returns {ChannelMate|null}
     */
    disconnect(mateId) {
        if (!(mateId in this.mates)) return null;
        this.disconnected[mateId] = this.mates[mateId];
        delete this.mates[mateId];
        return this.disconnected[mateId];
    }

    /**
     * @param {String} mateId 
     * @returns {ChannelMate|null}
     */
    reconnect(mateId) {
        if (!(mateId in this.disconnected)) return null;
        this.mates[mateId] = this.disconnected[mateId];
        delete this.disconnected[mateId];
        return this.mates[mateId];
    }

    /**
     * @param {String} mateId 
     * @returns {ChannelMate|null}
     */
    leave(mateId) {
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
        _sendData(this.socket, msg);
    }

    trigger(eventName, data = {}, receivers = null, returnToSender = true, privateMode = false) {
        let msg = _prepareMsg(this, 'event', data);
        msg.__metaData__.event = eventName;
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
 * Message
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class Message
{
    constructor(socket, conf) {
        this._socket = socket;
        this._data = conf.__lxws_message__;
    }

    getData() {
        return this._data;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ChannelMessage
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:namespace lx.socket;
class ChannelMessage extends lx.socket.Message {
    constructor(channel, conf) {
        super(channel.socket, conf);
        this.channel = channel;
        this._fromId = conf.from;
        this._addressed = conf.addressed;
        this._private = conf.private;
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
class ChannelEvent extends lx.socket.Message
{
    constructor(eventName, socket, params) {
        super(socket, params);

        this._name = eventName;
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

function _setSocketHandlerOnMessage(self) {
    if (self._socket === null) return;
    self._socket.onmessage =(e)=>{
        let msg = JSON.parse(e.data);

        if (self._id === null) {
            _onHandshake(self, msg);
            return;
        }

        if (msg.__lxws_message__) {
            if (!msg.channel) _runHandlers(self, new lx.socket.Message(self, msg), self._onMessage);
            else {
                const ch = self.getChannel(msg.channel);
                if (ch) {
                    const msgObj = new lx.socket.ChannelMessage(ch, msg);
                    _runHandlers(self, msgObj, self._onChannelMessage);
                    if (msg.channel in self._onChannel.message)
                        _runHandlers(self, msgObj, self._onChannel.message[msg.channel]);
                }
            }
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


        //TODO channel event
        //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
        //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
        //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
        //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
        //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
        //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
        // if (msg.__event__ && self._onEvent.len) {
        //     _processEvent(self, msg);
        //     return;
        // }
        // if (msg.__multipleEvents__ && self._onEvent.len) {
        //     msg.__multipleEvents__.forEach(e=>_processEvent(self, e));
        //     return;
        // }


        if (msg.__lxws_channel_event__) {
            let channel, mate;
            switch (msg.__lxws_channel_event__) {
                case 'mateEntered':
                    channel = self._channels.get(msg.channel);
                    if (!channel) return;
                    mate = channel.register(msg.id, msg.data);
                    if (!mate) return;
                    _runHandlers(self, 'channelMateEntered', self._onChannelMateEntered, {channel, mate});
                    if (msg.channel in self._onChannel.mateEntered)
                        _runHandlers(self, 'channelMateEntered',
                        self._onChannel.mateEntered[msg.channel], {channel, mate});
                    break;

                case 'mateUpdated':
                    channel = self._channels.get(msg.channel);
                    if (!channel) return;
                    mate = channel.getMate(msg.id);
                    if (!mate) return;
                    mate.setSharedData(msg.data);
                    _runHandlers(self, 'channelMateUpdated', self._onChannelMateUpdated, {channel, mate});
                    if (msg.channel in self._onChannel.mateUpdated)
                        _runHandlers(self, 'channelMateUpdated',
                        self._onChannel.mateUpdated[msg.channel], {channel, mate});
                    break;

                case 'mateLeft':
                    channel = self._channels.get(msg.channel);
                    if (!channel) return;
                    mate = channel.leave(msg.id);
                    if (!mate) return;
                    _runHandlers(self, 'channelMateLeave', self._onChannelMateLeft, {channel, mate});
                    if (msg.channel in self._onChannel.mateLeave)
                        _runHandlers(self, 'channelMateEntered',
                        self._onChannel.mateLeave[msg.channel], {channel, mate});
                    break;

                case 'mateDisconnected':
                    channel = self._channels.get(msg.channel);
                    if (!channel) return;
                    mate = channel.disconnect(msg.id);
                    if (!mate) return;
                    _runHandlers(self, 'channelMateDisconnected', self._onChannelMateDisconnected, {channel, mate});
                    if (msg.channel in self._onChannel.mateDisconnected)
                        _runHandlers(self, 'channelMateEntered',
                        self._onChannel.mateDisconnected[msg.channel], {channel, mate});
                    break;

                case 'mateReconnected':
                    channel = self._channels.get(msg.channel);
                    if (!channel) return;
                    mate = channel.reconnect(msg.id);
                    if (!mate) return;
                    _runHandlers(self, 'channelMateReconnected', self._onChannelMateReconnected, {channel, mate});
                    if (msg.channel in self._onChannel.mateReconnected)
                        _runHandlers(self, 'channelMateEntered',
                        self._onChannel.mateReconnected[msg.channel], {channel, mate});
                    break;
            }
            return;
        }

        if (msg.__lxws_action__) {
            switch (msg.__lxws_action__) {
                case 'connect':
                    if (msg.channel) {
                        const ch = msg.channel;
                        self._channels.create(ch.key, ch.data, ch.connections);                        
                    }
                    _runHandlers(self, 'connected', self._onConnected);
                    break;

                case 'reconnect':
                    self._id = msg.idRestored;
                    _updateReconnectId(self);
                    if (msg.channels)
                        msg.channels.forEach(ch=>self._channels.create(ch.key, ch.data, ch.connections));
                    _runHandlers(self, 'connected', self._onConnected);
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
            header[1].call(header[0], event);
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
    self._socket.onError =(e)=>{
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

function _processEvent(self, msg) {

    //TODO
    console.log('PROCESS EVENT!!!!');

    // let event = new lx.socket.ChannelEvent(msg.__event__, self, msg);
    // self._onEvent.forEach(handler => {
    //     if (handler instanceof lx.socket.EventListener)
    //         handler.processEvent(event);
    //     else if (lx.isFunction(handler))
    //         handler(event);
    // });
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
                event = new lx.socket.ConnectionEvent('beforeSend', this, data);
            if (handler(event) === false) return;
        }
    }

    if (!force && self._timer) self._buffer.push(data);
    else self._socket.send(JSON.stringify(data));
}
