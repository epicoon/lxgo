lx.EVENT_BEFORE_AJAX_REQUEST = 'beforeAjax';
lx.EVENT_AJAX_REQUEST_UNAUTHORIZED = 'ajaxUnauthorized';
lx.EVENT_AJAX_REQUEST_FORBIDDEN = 'ajaxForbidden';

let callbacks = {};

// @lx:namespace lx;
class Events extends lx.AppComponent {
    subscribe(eventName, callback) {
        if (!(eventName in callbacks))
            callbacks[eventName] = [];
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
