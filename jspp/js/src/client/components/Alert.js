let alerts = null;

function initAlerts() {
    let wrapper = lx.app.domSelector.getElementByAttrs({lxid: 'lx-alerts'});
    if (!wrapper) {
        wrapper = document.createElement('div');
        wrapper.setAttribute('lxid', 'lx-alerts');
        Object.assign(wrapper.style, {
            position: 'absolute',
            top: '0',
            left: '0',
            width: '100%',
            height: '100%',
        });
        const body = document.body;
        if (body.firstChild) body.insertBefore(wrapper, body.firstChild);
        else body.appendChild(wrapper);
    }

    alerts = lx.Box.rise(wrapper);
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
            callback: ()=>_print(msg)
        });
    }
}

function _print(msg) {
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
