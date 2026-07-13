# The package helps to work with the Web Socket protocol

> Actual version: `v0.1.0-alpha.3`. [Details](https://github.com/epicoon/lxgo/tree/master/ws/CHANGE_LOG.md)

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
    ReconnectionAllowed: true
    ReconnectionDuration: 5000
    LifecycleLog: true
    LifecycleError: true
```
.3


2. Using existed API

If you already have some resourse, for example `myHandlers.HelloHandler`, you can use it to handle WS-requests:

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


3. Using channels

//TODO


4. Connect from client side

- Common data
```js
// use the module with the WS-client:
// @lx:use lx.socket.WebSocketClient;

// Set up connection:
const socket = new lx.socket.WebSocketClient({
    protocol: 'ws',
    port: 8000,

    // There is a set of handlers triggered by different events:
    handlers: {
        onOpen:                    (event) => { /* when your connection opened */ },
        onConnected:               (event) => { /* when serser approved the connection */ },
        onClose:                   (event) => { /* when your connection closed */ },
        onError:                   (event) => { /* when error occured */ },
        onBeforeSend:              (event) => { /* before send data to server you can do something.
                                                   If handler returns {{false}} data will not be sent */ },
        onMessage:                 (event) => { /* when a message from server received */ },
        onChannelMessage:          (event) => { /* when a message in a channel received */ },
        onChannelMateEntered:      (event) => { /* a new connection in a channel */ },
        onChannelMateUpdated:      (event) => { /* when a connection shared data updated */ },
        onChannelMateLeft:         (event) => { /* a connection left a channel */ },
        onChannelMateDisconnected: (event) => { /* a connection was disconnected from a channel */ },
        onChannelMateReconnected:  (event) => { /* a connection was reconnected to a channel */ },

        //TODO not implemented yet!
        // onAccessDenied:            (event) => { /*  */ },
        // onChannelEvent:            () => { /*  */ },
    }
});

// Connect:
socket.connect();
```

- Handlers details
//TODO

- Alternative way to define handlers
//TODO


5. Widget `lx.socket.ChatBox`

The widget podives an opportunity to use a channel for chatting

```js
// use the module with the widget:
// @lx:use lx.socket.WebSocketClient;

// Create widget:
const chat = new lx.socket.ChatBox({
    key: 'chatBox',
    geom: [10, 10, 40, 40]
});
chat.setSocket(socket, 'channelName');
```
