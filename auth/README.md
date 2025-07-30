# Authentication microservice

There are ways to use the service:
* [Full ready-to-use solution for lxgo/kernel applications](#full-sol)
* [Ready-to-use solution for browser](#browser-sol)
* [Implementation of integration for server on the client side](#server-imp)
* [Implementation of integration for browser on the client side](#browser-imp)

Microservice API:
* [API](https://github.com/epicoon/lxgo/tree/master/auth/README_API.md)

<a name="full-sol"><h3>Full ready-to-use solution for lxgo/kernel applications</h3></a>

There is a [package](https://github.com/epicoon/lxgo/tree/master/auth_client) with already done solutiuons for integration. You can use it if your application based on `lxgo/kernel`.


<a name="browser-sol"><h3>Ready-to-use solution for browser</h3></a>

> Tokens should be kept in local storage with keys: `lxAuthAccessToken`, `lxAuthRefreshToken`

1. Plug in ready-to-use JS-bundle:
```html (twig)
<script defer src="{{ auth_server_url }}/js-client/bundle.js" onload="auth()"></script>
```
You'll get an object `lxAuth`:
* Events:
    - `lxAuth.TOKENS_FOUND` - if lxAuth can get tokens from the local storage
    - `lxAuth.TOKENS_NOT_FOUND` - if tokens are not in the local storage
    - `lxAuth.TOKENS_REMOVED` - after drop tokens from the local storage
* Methods:
    - `lxAuth.goToAuth()` - redirect to the auth server form
    - `lxAuth.logOut()` - log client out
    - `lxAuth.run()` - launching the Authentication Manager
    - `lxAuth.on(event, handler)` - subscribe to an lxAuth event
    - `lxAuth.async fetch(url, params = {})` - function-wrappet for `fetch` to ensure access token will be sent in auth header

2. The way to use `lxAuth` object:
```js
loginBut.addEventListener('mousedown', () => lxAuth.goToAuth());
logoutBut.addEventListener('mousedown', () => lxAuth.logOut());

lxAuth.on(lxAuth.TOKENS_NOT_FOUND, () => {
    // Display a button to allow login or an immediate redirect to the authorization service form depending on the needs of the application
});

lxAuth.on(lxAuth.TOKENS_FOUND, async () => {
    // Request to obtain user data
    const userData = await lxAuth.getUserData();
    if (userData.success) {
        // Further data using
    }
});

lxAuth.on(lxAuth.TOKENS_REMOVED, () => {
    // Show a button to allow login
});

// Launching the Authentication Manager
lxAuth.run();
```

3. To provide the Authentication Manager with access to the data it needs to operate, you need to assign a specific data structure (see "Data Structure" below) in JSON string format to the global variable `_lxauth_settings` (see the example below "Example of data preparation"). The Authentication Manager will read this data when it starts and delete the global variable.
Data Structure:
```yaml
authSettings:
    # Client application identifier
    id: 1
    # Ajax method to generate unique string for CSRF protection
    state_path: /gen-state
    # Endpoint for proxying request to /logout of authorization service
    logout_path: /logout
    # Endpoint for proxying request to /refresh of authorization service
    refresh_path: /auth-refresh
    # Endpoint for proxying request to /user-data of authorization service
    user_data_path: /get-user
    # URL of the authorization service
    server: http://auth_server_url
    # Endpoint for redirection from the authorization service form back to the client application
    redirect_uri: http://client_url/auth-callback
```
Example of data preparation
```html (twig)
<script>
    window._lxauth_settings = {{ authSettingsJSON }}
</script>
```


<a name="server-imp"><h3>Implementation of integration for server on the client side</h3></a>

Need to implement:
* Endpoint for redirection from the authorization service form back to the client application
    Request params from the Authentication Server:
    code  - string - unique string to exchange for tokens
    state - string - unique string for CSRF protection
    Steps:
    1. Check the returned state. It is assumed that it was previously saved (possibly in session)
    2. Exchange code for tokens (request to `/tokens`)
    3. Store tokens on the client (e.g. browsers local storage)
    4. If necessary (for web applications), redirect user to the desired page
    //TODO by anology with lxgo/auth_client/auth_callback_handler.go
* Endpoint for generating the unique string for CSRF protection
    //TODO by anology with lxgo/auth_client/state_handler.go
* Endpoint for proxying request to `/logout` of the authorization service
    //TODO by anology with lxgo/auth_client/logout_handler.go
* Endpoint for proxying request to /refresh of authorization service
    //TODO by anology with lxgo/auth_client/refresh_handler.go
* Endpoint for proxying request to /user-data of authorization service


<a name="browser-imp"><h3>Implementation of integration for browser on the client side</h3></a>

//TODO by analogy with lxgo/auth/client/js/apps/client/src/App.js
