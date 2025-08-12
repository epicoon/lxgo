## JS-application components

A **component** is a functional unit that extends the capabilities of an application at build time. It can provide additional logic, configuration, services, or reusable resources to the application or its elements. Unlike UI [elements](https://github.com/epicoon/lxgo/tree/master/jspp/doc/elements.md), components are not necessarily visual but serve as modular building blocks that can be plugged into the application’s structure to add or modify behavior.

* You can get access to **component** with code:
  ```js
  // Where "componentKey" is the key of a component
  lx.app.componentKey
  ```

* Component initialization. The best practice is initialization by configuration file. You can set the path to the file in the application configuration using key `Components.JSPreprocessor.AppConfig`. Alternative way is calling application method `start` in code and passing configuration object as parameter.
  - Example initialization by configuration file:
    ```yaml
    ```
  - Example initialization in code:
    ```js
    ```


## Built-in components

* [lx.CssManager](#comp1)
* [lx.ImageManager](#comp2)
* [lx.FunctionHelper](#comp3)
* [lx.Language](#comp4)
* [lx.Queues](#comp5)
* [lx.DomSelector](#comp6)
* [lx.Events](#comp7)
* [lx.Cookie](#comp8)
* [lx.Storage](#comp9)
* [lx.Binder](#comp10)
* [lx.Alert](#comp11)
* [lx.Tost](#comp12)
* [lx.Mouse](#comp13)
* [lx.Keyboard](#comp14)
* [lx.DragAndDrop](#comp15)
* [lx.Animation](#comp16)


### <a name="comp1">lx.CssManager</a>
- key: `cssManager`
- available: `client` `server`
#### Description
TODO
#### Initialization
- valiant 1
```yaml
TODO
```
- valiant 2
```yaml
TODO
```
- valiant 3
```yaml
TODO
```
#### Methods
TODO


### <a name="comp2">lx.ImageManager</a>
- key: `imageManager`
- available: `client` `server`

### <a name="comp3">lx.FunctionHelper</a>
- key: `functionHelper`
- available: `client` `server`

### <a name="comp4">lx.Language</a>
- key: `lang`
- available: `client` `server`

### <a name="comp5">lx.Queues</a>
- key: `queues`
- available: `client`

### <a name="comp6">lx.DomSelector</a>
- key: `domSelector`
- available: `client`

### <a name="comp7">lx.Events</a>
- key: `events`
- available: `client`

### <a name="comp8">lx.Cookie</a>
- key: `cookie`
- available: `client`

### <a name="comp9">lx.Storage</a>
- key: `storage`
- available: `client`

### <a name="comp10">lx.Binder</a>
- key: `binder`
- available: `client`

### <a name="comp11">lx.Alert</a>
- key: `alert`
- available: `client`

### <a name="comp12">lx.Tost</a>
- key: `tost`
- available: `client`

### <a name="comp13">lx.Mouse</a>
- key: `mouse`
- available: `client`

### <a name="comp14">lx.Keyboard</a>
- key: `keyboard`
- available: `client`

### <a name="comp15">lx.DragAndDrop</a>
- key: `dragAndDrop`
- available: `client`

### <a name="comp16">lx.Animation</a>
- key: `animation`
- available: `client`


## Add custom component



