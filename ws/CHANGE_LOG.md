------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.7
Changes:
- fix: `Connection`'s own fields (status, channels, shared data, the underlying `net.Conn`, `isReadyToClose`, etc.)
  were read/written from multiple goroutines without synchronization - e.g. one connection's channel broadcast
  could read a mate's `Status`/call its `Send` concurrently with that connection's own goroutine writing them - now
  guarded by a `sync.RWMutex`
- fix: `ConnRepo.Reconnect` mutated the connections map under a read lock (a data race) and could deadlock by
  holding one lock while re-entering the repo through a channel broadcast - rewritten to take the write lock for
  mutations and release every lock before calling out
- fix: `ConnRepo.GetAll()` returned the live internal map instead of a copy, letting callers race with concurrent
  connects/disconnects
- fix: `component.WSServer.listener` raced between `Start()` and `Stop()` (different goroutines) - guarded with a
  mutex; `Stop()` no longer errors out if called before `Start()` ever ran
- fix: go.mod was missing its `require` block entirely - the module could not be resolved/built standalone outside
  this monorepo's `go.work`
- test: unit and integration tests across the package (channels, connections, connection/channel repos, router,
  component)

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.27
Version: v0.1.0-alpha.6
Changes:
- internal: `internal/src.Connection`/`Router` adapted to `lxgo-kernel`'s `conv`→`cast` package rename and its
  `kernel.NewData(...)` removal (now `kernel.Dict{...}` literals) - no change to `ws`'s own public API

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.5
Changes:
- docs: Go-doc comments for every exported declaration in the root package (`IWSServer`, `IConnRepo`/`IConnection`,
  `IChannelRepo`/`IChannel`, `IChannelBuilder`, `IMessage`/`IChannelMessage`, `IChannelEvent`, `ChannelCloseCode*`)
  and the `component` subpackage - previously undocumented

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.4
Changes:
- add: multi-channel support - socket.enterChannel(key, sharedData?)/leaveChannel(key) let a connection dynamically join/leave any server-created channel, not just the configured default one
- add: channel-wide custom events - channel.trigger()/onChannelEvent, with server-side adjudication via IChannel.SetEventHandler (inspect/mutate the data, narrow receivers, or suppress delivery before it's relayed)
- add: onActionError - malformed/unknown actions (and channel-entry/creation rejections) now get an explicit error response instead of being silently dropped
- add: Origin validation (Components.WSServer.AllowedOrigins) - the first thing that actually sends the previously-dead WS close code 1002/onAccessDenied
- add: channel entry authorization - IChannel.SetAuthHandler lets application code gate enterChannel (password, invite-list, whatever)
- add: client-initiated channel creation - socket.createChannel(public, proprietary, sharedData?, initData?); the server always generates the key; ChannelValidator/ChannelCreatedHandler hooks at the component level
- add: public channel discovery - connect()/reconnect() surface every public channel (plus any private one already joined) in a unified "channels" list, live-updated via onChannelCreated as new ones appear
- add: MaxChannelsPerConnection - per-connection cap on client-created channels, freed when a created channel closes, preserved across reconnect
- add: channel closing - a proprietary channel closes when its creator leaves for good; a non-proprietary one auto-closes after sitting empty past EmptyChannelTTL; IChannel.Close(code) for explicit server-side closing either way; onChannelClosed on the client
- fix: WSServer.Start()'s accept loop no longer spins in a tight infinite error loop once Stop() closes the listener
- fix: Channel.trigger() didn't actually apply receivers/returnToSender/privateMode to the outgoing message (only send() did)
- fix: several dead client-side handler bugs (wrong onerror casing, a tuple-handler dispatch typo, ChatBox.js's toast method name typo)
- refactor: ChatBox.js migrated from @lx:use directives to lx.import(...) calls
- remove: unused Protocol config field (wss/TLS support was scoped out of this package - terminate TLS at a reverse proxy instead)

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.13
Version: v0.1.0-alpha.3
Changes:
- fix: unify error logging for Send()/Close() calls across the connection lifecycle (via LifecycleLog/LifecycleError), previously silently ignored

------------------------------------------------------------------------------------------------------------------------
Date: 2026.03.19
Version: v0.1.0-alpha.2
Changes:
- refactor lx.socket.WebSocketClient

------------------------------------------------------------------------------------------------------------------------
Date: 2025.12.21
Version: v0.1.0-alpha.1
