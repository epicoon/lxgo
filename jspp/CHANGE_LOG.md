------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.09
Version: v0.1.0-alpha.9
Changes:
- added lxml comments

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.03
Version: v0.1.0-alpha.8
Changes:
- fix lxml parsing

------------------------------------------------------------------------------------------------------------------------
Date: 2025.09.02
Version: v0.1.0-alpha.7
Changes:
- fix lxml parsing with spaces

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.20
Version: v0.1.0-alpha.6
Changes:
- refactor Dialog.js

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.15
Version: v0.1.0-alpha.5
Changes:
- add feature plugin ajax-requests routing
- fix lxml html content for empty widget

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.14
Version: v0.1.0-alpha.4
Changes:
- fix plugin config with map for images
- changed syntax:
    - lx(i18n).key  ->  lx.i18n(key)
    - lx(STATIC).CONST  ->  lx.self(CONST)
- added feature for translations with params, example:
    in tr.yaml:
        key: text with ${param}
    in code:
        lx.i18n(key, {param: value})

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.13
Version: v0.1.0-alpha.3
Changes:
- fixed main plugin code internationalization

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.12
Version: v0.1.0-alpha.2
Changes:
- fixed rendering plugin as page without template
- added part of documentation
- fixes in JS-code

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
