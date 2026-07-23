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
