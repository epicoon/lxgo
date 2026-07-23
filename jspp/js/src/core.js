(()=>{
    const lx = { app: null };
    Object.defineProperty(lx, 'globalContext', {
        get: function () {
            // @lx:<context CLIENT:
            return window;
            // @lx:context>
            // @lx:<context SERVER:
            return global;
            // @lx:context>
        }
    });
    lx.globalContext.lx = lx;

    lx.import(
        'common/js_extends',
        'common/lx_core',
        'Application',
        '-R common/tools/'
    );
    // @lx:<context CLIENT:
        lx.import(
            'client/tools/behavior/',
            'client/tools/request/',
            'client/tools/'
        );
    // @lx:context>
    // @lx:<context SERVER:
        lx.import('-R server/tools/');
    // @lx:context>
    lx.import(
        'common/widgets/Rect/Rect',
        'common/widgets/TextBox',
        'common/widgets/Box/Box'
    );

    lx.app = new lx.Application();
})();
