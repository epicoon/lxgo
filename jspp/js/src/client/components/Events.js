lx.EVENT_BEFORE_AJAX_REQUEST = 'beforeAjax';
lx.EVENT_AJAX_REQUEST_UNAUTHORIZED = 'ajaxUnauthorized';
lx.EVENT_AJAX_REQUEST_FORBIDDEN = 'ajaxForbidden';

let callbacks = {};

// @lx:namespace lx;
class Events extends lx.AppComponent {
    /**
     * @param {String} eventName
     * @param {Function} callback
     */
    on(eventName, callback) {
        if (!(eventName in callbacks))
            callbacks[eventName] = [];
        callbacks[eventName].push(callback);
    }

    /**
     * @param {String} eventName 
     * @param {Function} callback 
     * @returns 
     */
    deny(eventName, callback) {
        if (!(eventName in callbacks)) return;
        callbacks[eventName].lxRemove(callback);
    }

    /**
     * @param {String} eventName
     * @param {any} data
     * @returns 
     */
    trigger(eventName, data = {}) {
        if (!(eventName in callbacks)) return;
        callbacks[eventName].forEach(callback=>callback(new lx.Event(eventName, data)));
    }
}

// @lx:namespace lx;
class Event {
	constructor(name, data = {}) {
		this.name = name;
		this.data = data;
	}

	getData() {
		return this.data;
	}
}
