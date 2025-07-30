lx.EVENT_BEFORE_AJAX_REQUEST = 'beforeAjax';
lx.EVENT_AJAX_REQUEST_UNAUTHORIZED = 'ajaxUnauthorized';
lx.EVENT_AJAX_REQUEST_FORBIDDEN = 'ajaxForbidden';

let callbacks = {
    beforeAjax: [],
    ajaxUnauthorized: [],
    ajaxForbidden: []
};

// @lx:namespace lx;
class LifeCycle extends lx.AppComponent {
    subscribe(eventName, callback) {
        if (!(eventName in callbacks)) return;
        callbacks[eventName].push(callback);
    }

    deny(eventName, callback) {
        if (!(eventName in callbacks)) return;
        callbacks[eventName].lxRemove(callback);
    }

    trigger(eventName, params = []) {
        if (!(eventName in callbacks)) return;
        callbacks[eventName].forEach(callback=>callback.apply(null, params));
    }
}
