# Preprocessor features

## Directive `@config`: forwarding an backend-application configuration parameter to frontend-application:
    If you have a parameter in your `config.yaml` file:

    ```yaml
    Params:
      Param: 1
    ```

    You can use it in the JS-application configuration file:

    ```yaml
    params:
      paramFromBackendOnFrontend: '@config(Params.Param)'
    ```

# TODO
Preprocessor features
- lx.i18n(
- lx.static.
- require
- ...
