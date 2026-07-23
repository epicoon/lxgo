# The package helps to work with the Web Socket protocol

> Actual version: `v0.1.0-alpha.4`. [Details](https://github.com/epicoon/lxgo/tree/master/ws/CHANGE_LOG.md)

> You can use it if your application is based on [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel)


Features:
* Web Socket server and client
* Opportunity to use standard HTTP-resources: `[http.Resource](https://github.com/epicoon/lxgo/tree/master/kernel/http/resource.go)`
* Multi-channel architecture
* Chat widget


1. Setup component

- Add component to application
```go
package main

import (
    // ...
	wsComp "github.com/epicoon/lxgo/ws/component"
)

// ...

func setComponents(app kernel.IApp) error {
    // ...

    // Components.WSServer - config path
	if err := wsComp.SetAppComponent(app, "Components.WSServer"); err != nil {
		return err
	}
}
```

- Setup component config
```yaml
Components:
  WSServer:
    Host: localhost
    Port: 8100
    DefaultChannel:
      Key: channel
      SharedData:
        Name: channel
    MaxRequestsPerMinute: 10
    MaxConnectionsPerIp: 5
    MaxChannelsPerConnection: 3
    EmptyChannelTTL: 300
    AllowedOrigins:
      - https://example.com
    ReconnectionAllowed: true
    ReconnectionDuration: 5000
    LifecycleLog: true
    LifecycleError: true
```

`AllowedOrigins` is optional - empty/unset means no restriction (any `Origin` is accepted, including a
missing header). Set it to restrict WS connections to a specific list of origins; a handshake with a
non-matching (or missing) `Origin` header is upgraded and then immediately closed with WS close code
`1002` - see `onAccessDenied` in "Handlers details" below.

`MaxChannelsPerConnection` is optional - 0/unset means no limit. It only counts channels a connection
created itself via `socket.createChannel(...)` (see "Using channels" below), not channels it merely
joined; the quota is freed again once a created channel closes (see "Channel closing" below) - not
before, so it isn't released just because the creator temporarily disconnects (only survives across
disconnect/reconnect together with the rest of that connection's state).

`EmptyChannelTTL` is optional (seconds) - 0/unset disables it entirely. See "Channel closing" below.


2. Using the existing API

If you already have some resource, for example `myHandlers.HelloHandler`, you can use it to handle WS-requests:

```go
package main

import (
    // ...
	"github.com/epicoon/lxgo/kernel"
    wsComp "github.com/epicoon/lxgo/ws/component"
)

// ...

// app - your kernel.IApp application
// myHandlers.NewHelloHandler - kernel.CHttpResource
ws, _ := wsComp.AppComponent(app)
ws.Router().RegisterResources(kernel.HttpResourcesList{
    "/hello": myHandlers.NewHelloHandler,
})
```

//TODO - надо раскрыть, как это со стороны клиента использовать


3. Using channels

A channel is a named, server-tracked group of connections that receive broadcast messages and membership-lifecycle
events together (`ws.IChannel`/`ChannelRepo` on the Go side).

- **The default channel.** If `Components.WSServer.DefaultChannel.Key` is set in the config (see "Setup component"
  above), that one channel is created automatically when the server starts, and **every connection joins it
  automatically** on `connect`/`reconnect` - there's nothing to do on either side to make this happen.
- **Other channels.** A channel can be created either from the server side or by a client:
  - Server side: `ws.Channels().CreateChannel(builder)` (`ws` here is the `*component.WSServer` you got from
    `wsComp.AppComponent(app)`; `builder` is `ws.NewChannelBuilder().SetKey(key).SetSharedData(data)...` - see
    "Client-created channels" below for the builder's other fields) returns `(ws.IChannel, string)` - `nil` and a
    reason on failure (key collision, or rejected by `ChannelValidator` - see below).
  - Client side: `socket.createChannel(public, sharedData?, initData?)` - see "Client-created channels" below.

  Once a channel exists (either way), a connection joins it client-side: `socket.enterChannel(channelKey,
  sharedData?)` - once the server acknowledges (`onChannelEntered`, see "Handlers details" below), the channel
  becomes available via `socket.getChannel(channelKey)` and behaves exactly like the default channel (same
  membership events, same `channel.send(...)`). Leave with `socket.leaveChannel(channelKey)` (acknowledged via
  `onChannelLeft`). A connection can be in any number of channels at once, including zero (the default channel is
  just the one you're auto-entered into, nothing more special about it) - entering/leaving one doesn't affect the
  others.
- **Client-created channels.** `socket.createChannel(public, proprietary, sharedData?, initData?)` - the server
  always generates the channel key itself (there's no client-supplied key, so different clients can never collide
  on one) and the creator is automatically entered into it on success, same as a manual `enterChannel` would
  (acknowledged via `onChannelCreated`, see "Handlers details" below - the response carries the server-generated
  key). `public` controls whether the channel's key is later announced to other connections (see "Public channels"
  below); `proprietary` controls how it closes (see "Channel closing" below). `initData` is never stored on the
  channel or sent to anyone - it only exists to be inspected by `ChannelValidator`/`ChannelCreatedHandler` (Go, see
  below), e.g. to carry a password that a validator wires into `SetAuthHandler` for the new channel. A rejected
  `createChannel` (limit exceeded, denied by `ChannelValidator`, or a wrong type for
  `public`/`proprietary`/`sharedData`/`initData`) comes back as an `onActionError`, same convention as `enterChannel`.
- **Channel closing.** There's no client-side "close channel" action, but a channel can close in three ways -
  two automatic (chosen at creation time via `proprietary` - client-side `createChannel` - or
  `builder.SetProprietary(true)` server-side), one explicit and Go-only:
  - **Proprietary** (`ws.IChannel.CreatorID()` is meaningful): closes the moment its creator leaves it for
    good - either explicitly (`leaveChannel`), or by disconnecting and never reconnecting (a temporary drop that's
    still eligible to reconnect does *not* close it).
  - **Non-proprietary**: has no owner in that sense - instead it closes itself automatically once it's been
    completely empty for `Components.WSServer.EmptyChannelTTL` seconds straight (disabled/never auto-closes if
    that's 0/unset) - checked roughly once a second, so actual closure can lag up to ~1s past the configured TTL.
    The `DefaultChannel` is never eligible for this, regardless of `EmptyChannelTTL`.
  - **Explicit (Go only)**: application code can force-close any channel at any time with
    `ws.IChannel.Close(code string)` - e.g. `ws.Channels().Get(key).Close("game_over")`, or on a reference kept
    around from `ChannelCreatedHandler`. Kicks everyone currently in it (the creator too, if still a member).
    Safe to call more than once - a no-op after the first.

  Whichever way it happens, every connection still in the channel at that point gets `onChannelClosed`
  (`{channelKey, code}`) - already removed from `getChannel()`/`getAvailableChannels()` by the time it fires - and
  the creator's `MaxChannelsPerConnection` quota (if any) is freed. `code` is relayed to the client as-is, meant to
  let it show a situation-appropriate message: `ws.ChannelCloseCodeCreatorGone` (`"200"`) for the
  proprietary-creator-departure case, `ws.ChannelCloseCodeServer` (`"100"`) for the empty-channel sweep or an
  explicit `Close(...)` call that didn't pass anything more specific - application code is free to pass any other
  string of its own to `Close(...)` for its own reasons (e.g. `"game_over"`).
- **Public channels & discovering what's available.** A channel created with `public: true` (client-side) or
  `builder.SetPublic(true)` (server-side) is discoverable without already knowing its key - every `connect`/
  `reconnect` response includes every public channel (server-wide) plus any *private* channel the connecting
  connection happens to already be a member of (e.g. its restored memberships on `reconnect`, or the default
  channel it was just auto-entered into on `connect`) in `data.channels`: `{key, data[, connections]}` - `data` is
  the channel's shared data (e.g. a human-readable name/icon/type). `socket.getAvailableChannels()` mirrors this
  client-side as `[{key, data}, ...]` and updates live afterwards via `onChannelCreated` (see "Handlers details" below)
  whenever *any* new channel becomes known to this connection - your own `createChannel()` succeeding, or someone else's
  public one appearing - so no one has to reconnect to learn about a new room. A private channel created by someone
  else isn't announced anywhere unless this connection is/becomes a member of it - a connection needs to already
  know its key (from application logic outside this package) to call `enterChannel` on it.
- **Channel creation hooks (Go only, component-level - set once, apply to every channel created afterwards).**
  `ws.IWSServer.SetChannelValidator(func(b ws.IChannelBuilder) (bool, string) {...})` runs for every
  `CreateChannel` call (client or server-initiated - `b.Creator()` is `nil` for the latter), *except* the automatic
  `DefaultChannel` at startup, which always skips it; returning `false` rejects creation with the given reason.
  `ws.IWSServer.SetChannelCreatedHandler(func(ch ws.IChannel, initData map[string]any) {...})` runs once for every
  successfully created channel, *including* `DefaultChannel` (which always gets an empty `initData`, since it isn't
  a creation request from anyone) - this is the place to call `ch.SetEventHandler(...)`/`ch.SetAuthHandler(...)` on
  it right after creation, uniformly, regardless of who created it. Both are nil by default (validator allows
  everything; created-handler does nothing extra) - set them before calling `Start()`, since `DefaultChannel` is
  created at the very start of `Start()` (not earlier), specifically so `ChannelCreatedHandler` is guaranteed to
  already be registered by the time it fires for it.
- **Entry authorization.** By default any connection can enter any channel it knows the key of (including the
  default channel, automatically). To restrict entry: `ws.IChannel.SetAuthHandler(func(conn ws.IConnection, message
  map[string]any) (bool, string) {...})` (Go, set right after `CreateChannel`) runs once per connection, the first
  time it tries to enter that channel - `message` is the same map the client sent (`enterChannel`'s optional fields,
  e.g. a `password` key sent alongside `channelKey`/`sharedData`), so a password (or any other applicaton-defined
  check) travels the same way `sharedData` already does. Returning `false` rejects entry with the given reason
  string; the client sees it as an explicit `onActionError` (`{action: "enterChannel", error: "<reason>"}`).
  For the default channel specifically, a rejection doesn't fail the `connect`/`reconnect` handshake itself -
  the connection is still established, it just won't be a member of that channel (no `channel` key in the `connect`
  response). No handler set (the default) means "allow everyone".
- **Shared data.** A connection's `getSharedData()` (client-side, see below) becomes its baseline shared data; a
  channel can layer additional per-channel data on top via `channel.updateSharedData(data)`. Other members see it as
  a `ChannelMate`'s properties (`mate.someKey`, mirroring whatever keys `data` had).
- **Sending a channel message**: `channel.send(data, receivers = null, returnToSender = false, privateMode = false)`
  - omit `receivers` to broadcast to the whole channel. Delivered to `onChannelMessage` (see "Handlers details"
    below) as a `ChannelMessage`.
- **Membership events** fire automatically for every mate whenever someone joins/updates their shared data/leaves/
  disconnects/reconnects - see `onChannelMate*` in "Handlers details" below; no setup needed beyond registering the
  handler.
- **Channel-wide custom events**: `channel.trigger(eventName, data = {}, receivers = null, returnToSender = true,
  privateMode = false)` - same delivery options as `send()` (see above), just a named event instead of a bare
  message. Received via `onChannelEvent` (see "Handlers details" below) as a `ChannelEvent`.
- **Server-side event adjudication.** Unlike a plain `channel.send()` message (always relayed as-is), a triggered
  event can be intercepted server-side: `ws.IChannel.SetEventHandler(func(event ws.IChannelEvent) {...})` (Go, set
  right after `CreateChannel`) runs for every event on that channel *before* it's relayed - the handler can inspect
  `event.Name()`/`event.Data()`/`event.Initiator()`, mutate the data (`event.SetData(...)`/
  `event.SetDataForConnection(...)`), narrow who receives it (`event.SetReceiverIds(...)`), or call `event.Stop()` to
  suppress delivery entirely (not even to the sender) - e.g. to validate a server-side logic and only relay the
  server-confirmed result. No handler set (the default) means "just relay it", same as a plain message.


4. Connect from client side

- Common data
```js
// use the module with the WS-client:
lx.import(lx.socket.WebSocketClient);

// Set up connection:
const socket = new lx.socket.WebSocketClient({
    protocol: 'ws',
    port: 8000,

    // There is a set of handlers triggered by different events:
    handlers: {
        onOpen:                    (event) => { /* when your connection opened */ },
        onConnected:               (event) => { /* when server approved the connection */ },
        onActionError:             (event) => { /* when the server rejected an action - connect/reconnect/
                                                   enterChannel/leaveChannel/createChannel - with an explicit error */ },
        onAccessDenied:            (event) => { /* when the server closed the connection with WS close code 1002 */ },
        onClose:                   (event) => { /* when your connection closed (any other close code) */ },
        onError:                   (event) => { /* when error occured */ },
        onBeforeSend:              (event) => { /* before send data to server you can do something.
                                                   If handler returns {{false}} data will not be sent */ },
        onMessage:                 (event) => { /* when a message from server received */ },
        onChannelCreated:          (event) => { /* a new channel became known - your own createChannel(), or
                                                   someone else's public one */ },
        onChannelEntered:          (event) => { /* when enterChannel() was acknowledged by the server */ },
        onChannelLeft:             (event) => { /* when leaveChannel() was acknowledged by the server */ },
        onChannelClosed:           (event) => { /* a channel you were in got closed server-side - see "Channel closing" */ },
        onChannelMessage:          (event) => { /* when a message in a channel received */ },
        onChannelMateEntered:      (event) => { /* a new connection in a channel */ },
        onChannelMateUpdated:      (event) => { /* when a connection shared data updated */ },
        onChannelMateLeft:         (event) => { /* a connection left a channel */ },
        onChannelMateDisconnected: (event) => { /* a connection was disconnected from a channel */ },
        onChannelMateReconnected:  (event) => { /* a connection was reconnected to a channel */ },
        onChannelEvent:            (event) => { /* a named event triggered in a channel - see "Handlers details" below */ },
    }
});

// Connect:
socket.connect();
```

- Handlers details

  What each handler actually receives:
  * `onOpen`, `onConnected`, `onActionError`, `onChannelCreated`/`onChannelEntered`/`onChannelLeft` and the five
    `onChannelMate*` handlers get a `ConnectionEvent` - `event.getName()` (the event name) and `event.getData()`
    (`{}` for `onOpen`/`onConnected`; `{action, error}` (both strings) for `onActionError`; `{key, data}` for
    `onChannelCreated`; `{channel}` - a `lx.socket.Channel` - for `onChannelEntered`; `{channelKey}` for
    `onChannelLeft`; `{channel, mate}` - `Channel`/`ChannelMate` instances - for the `onChannelMate*` ones).
  * `onActionError` fires for **any** rejected action - unknown `__lxws_action__` value, a malformed field (e.g. a
    non-string `channelKey`), an `enterChannel`/`leaveChannel` for a channel key the server never created, an
    `enterChannel` rejected by that channel's `SetAuthHandler` (see "Entry authorization" above), or a `createChannel`
    rejected by `ChannelValidator` or the `MaxChannelsPerConnection` limit (see "Client-created channels" above).
    Every rejection comes back as `{"__lxws_action__": "<attempted action>", "error": "<message>"}` (`action` is `""`
    if the action name itself couldn't be read). A malformed `reconnect` still falls back to a fresh `connect()`
    afterwards, `onActionError` fires first, then the ordinary `onConnected`.
  * `onAccessDenied`/`onClose`/`onError` get the **raw browser event** as-is (a `CloseEvent` for the first two, since
    they share the same underlying `WebSocket.onclose` handler and are only told apart by `event.code === 1002`;
    whatever `onerror` gives you for the third) - not wrapped in `ConnectionEvent`.
    Currently the only thing that triggers a real `1002` is a failed `Origin` check (see `AllowedOrigins` in "Setup
    component" above) - the handshake completes (`101 Switching Protocols`) and the connection is immediately closed
    with that code, so the check happens after upgrade rather than as an HTTP-level rejection (a raw HTTP failure
    wouldn't carry a meaningful `CloseEvent.code` in a browser). Channel-entry authorization (see "Entry authorization"
    above) is a separate, per-channel decision - a rejection there is an `onActionError`, not a connection-level
    close, so it doesn't trigger `onAccessDenied`. Connection-level authorization on `connect`/`reconnect` themselves
    is still `//TODO auth` (`internal/src/connection.go`) and doesn't trigger this yet either.
  * `onBeforeSend` gets a `ConnectionEvent` too, but with `event.getData()` set to the outgoing message object itself
    (about to be sent) - return `false` from the handler to cancel that send.
  * `onChannelCreated` fires whenever a new channel becomes known to this connection - either your own
    `createChannel()` was just acknowledged (`event.getData().key` is the server-generated key - there's no way to
    know it beforehand; the creator is already auto-entered by this point, `socket.getChannel(key)` is usable), or a
    *different* connection created a new public channel (this connection hasn't entered it, just learned it exists -
    `enterChannel(key)` still needed to actually join; never fires for someone else's *private* channel). Either
    way `event.getData()` is `{key, data}` - whether you're a member isn't part of the event itself, check
    `socket.getChannel(key)` (non-`null` only if you are) if the distinction matters to your code. There's no
    separate `onChannelEntered` round-trip for the creator's own case. `onChannelEntered`
    fires once a (non-creating) `enterChannel()` is acknowledged - by then `event.getData().channel` (and
    `socket.getChannel(channelKey)`) are already usable, same as after `connect()`'s automatic default-channel join.
    `onChannelLeft` fires once `leaveChannel()` is acknowledged, **including** when it was a no-op (the connection
    wasn't a member of that channel to begin with) - see "Using channels" above for why the ack always happens.
    `onChannelClosed` fires when the server closes a channel this connection was in from under it (see "Channel
    closing" above) - `event.getData()` is `{channelKey, code}` (`code` is a plain string - one of this package's
    own `ws.ChannelCloseCode*` values, or an application-chosen one, see "Channel closing"); unlike the others,
    this isn't a response to anything *this* connection did. Accepts an optional leading `channelKey` (like
    `onChannelMessage`) to scope it to one channel - see "Alternative way to define handlers" below.
  * `onMessage` gets a `Message` (`event.getData()`).
  * `onChannelMessage` gets a `ChannelMessage` (extends `Message`): `event.getData()`, `event.getAuthor()` (→ the
    sending `ChannelMate`), `event.isPrivate()`, `event.isAddressed()`, `event.isFromMe()`.
  * `onChannelEvent` gets a `ChannelEvent` (extends `ChannelMessage` - same `getData()`/`getAuthor()`/`isPrivate()`/
    `isAddressed()`/`isFromMe()`, plus `getName()` for the triggered event's name). The callback can be a plain
    function, a `[thisObj, function]` tuple, or an `lx.socket.EventListener` instance - the latter gets its
    `.processEvent()` called instead of being invoked directly, which dispatches by event name to a method named
    `on<EventName>` (camelCased), falling back to `.onEvent()` if there's no matching method. Like
    `onChannelMessage`/`onChannelMate*`, `onChannelEvent(channelKey, callback)` also accepts a leading channel key to
    scope it to one channel.

- Alternative way to define handlers

  The `handlers: {...}` block above is a shortcut for calling the same-named method after construction - e.g.
  `handlers: { onMessage: fn }` is equivalent to `socket.onMessage(fn)`. Calling the method form directly gives you
  two things the constructor shortcut can't:
  1. **Multiple handlers for the same event** - each `socket.onXxx(fn)` call *adds* a handler rather than replacing
     one, so you can call it more than once.
  2. **Scoping a channel handler to one specific channel.** The eight channel handlers
     (`onChannelMessage`/`onChannelEvent`/`onChannelMateEntered`/`onChannelMateUpdated`/`onChannelMateLeft`/
     `onChannelMateDisconnected`/`onChannelMateReconnected`/`onChannelClosed`) accept an optional leading `channelKey` argument:
     `socket.onChannelMessage(channelKey, fn)` only fires for events on that one channel, instead of every channel
     the connection is in. This can't be expressed through the constructor's `handlers: {...}` map at all - it's
     only reachable by calling the method directly. `lx.socket.ChatBox` itself does exactly this internally
     (`socket.onChannelMessage(this.channel, ...)`), see below.

  Any handler (in either form) can also be a `[thisObj, function]` tuple instead of a plain function, to bind `this`
  inside the callback without a wrapper closure - e.g. `onMessage: [myWidget, myWidget.handleMessage]`.


5. Widget `lx.socket.ChatBox`

The widget provides an opportunity to use a channel for chatting

```js
// use the module with the widget:
lx.import(lx.socket.WebSocketClient);

// Create widget:
const chat = new lx.socket.ChatBox({
    key: 'chatBox',
    geom: [10, 10, 40, 40]
});
// 'channelName' has to be a channel the connection is actually in - the default one
// (Components.WSServer.DefaultChannel.Key from your config, see "Setup component" above) unless
// you've wired up joining other channels yourself, see "Using channels" above.
chat.setSocket(socket, 'channelName');
```


## License

Apache License 2.0 — see [LICENSE](./LICENSE).
