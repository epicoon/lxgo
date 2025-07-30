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

    // @lx:require common/js_extends;
    // @lx:require common/lx_core;
    // @lx:require Application;
    // @lx:require -R common/tools/;
    // @lx:<context CLIENT:
        // @lx:require client/tools/behavior/;
        // @lx:require client/tools/request/;
        // @lx:require client/tools/;
    // @lx:context>
    // @lx:<context SERVER:
        // @lx:require -R server/tools/;
    // @lx:context>
    // @lx:require common/widgets/Rect/Rect;
    // @lx:require common/widgets/TextBox;
    // @lx:require common/widgets/Box/Box;

    lx.app = new lx.Application();
})();
