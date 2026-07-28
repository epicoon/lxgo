# lxgo

`lxgo` is a lightweight, modular Go web-framework ecosystem built around [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel) -
a small application core (built directly on the standard `net/http`, no third-party web framework underneath) that
gives you a component system, YAML configuration, routing and request handling. Everything else in this repository
is an optional add-on you pull in only if your application needs it: HTTP sessions, a thin DB layer over
[GORM](https://gorm.io/), WebSocket channels, a JS preprocessor with its own widget set for building a frontend
without a separate JS build toolchain, CLI command scaffolding, DB migrations, and a standalone OAuth2-like
authentication microservice with a ready client adapter.

This repository contains several `Go` packages:

* [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel) - web-server framework
* [lxgo/session](https://github.com/epicoon/lxgo/tree/master/session) - if you need HTTP sessions for your web-application
* [lxgo/query](https://github.com/epicoon/lxgo/tree/master/query) - if you need to work with a DB
* [lxgo/ws](https://github.com/epicoon/lxgo/tree/master/ws) - if you need WebSocket support for your web-application
* [lxgo/cmd](https://github.com/epicoon/lxgo/tree/master/cmd) - tool helps to create console commands
* [lxgo/migrator](https://github.com/epicoon/lxgo/tree/master/migrator) - tool to manage DB migrations
* [lxgo/jspp](https://github.com/epicoon/lxgo/tree/master/jspp) - javascript preprocessor useful for web-application frontend developing

Also:
* [lxgo/auth](https://github.com/epicoon/lxgo/tree/master/auth) - authentication microservice
* [lxgo/auth_client](https://github.com/epicoon/lxgo/tree/master/auth_client) - client adapter for the previous one

> Every package has its own `README.md` file and `CHANGE_LOG.md` file where you can check actual version of the package

> Use packages by importing in your project: `import "github.com/epicoon/lxgo/{pkg-name}"`  
> and run `go mod tidy`

> While package using you'll find in you `go.mod` file: `require github.com/epicoon/lxgo/{pkg-name} v{actual-version}`

## License

Every package is distributed under the Apache License 2.0 - see [LICENSE](./LICENSE).
