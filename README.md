# lxgo

This repository contains several `Go` packages:

* [lxgo/kernel](https://github.com/epicoon/lxgo/tree/master/kernel) - web-server framework
* [lxgo/session](https://github.com/epicoon/lxgo/tree/master/session) - if you need HTTP sessions for your web-application
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
