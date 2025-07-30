let alerts = null;

function initAlerts() {
    alerts = lx.Box.rise(lx.app.domSelector.getAlertsElement());
    alerts.key = 'alerts';
}

// @lx:namespace lx;
class Alert extends lx.AppComponent {
    init() {
        lx.alert = msg => this.print(msg);
    }

    print(msg) {
        if (!alerts) initAlerts();
        lx.app.dependencies.promiseModules({
            modules: ['lx.ActiveBox'],
            callback: ()=>__print(msg)
        });
    }
}

function __print(msg) {
    var el = new lx.ActiveBox({
        parent: alerts,
        geom: [10, 5, 80, 80],
        depthCluster: lx.DepthClusterMap.CLUSTER_URGENT,
        key: 'lx_alert',
        header: 'Alert',
        closeButton: {click: function(){this.parent.parent.del();}}
    });
    el.overflow('auto');
    el.get('body').html('<pre>' + msg + '</pre>');
}
