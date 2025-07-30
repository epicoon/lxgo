let guid = 0;
let appointEvent = Function("var a=arguments;a[0].handle=function(e){return a[1].call(a[0],e);}");

// @lx:namespace lx;
class DomEvent extends lx.AppComponent {
    appoint(el, events = {}) {
        if (el.events === undefined) {
            el.events = events;
            appointEvent(el, commonHandle);
            for (var type in el.events) addEventListener(el, type);
        }
    }

    disappoint(el) {
        if (el.events === undefined) return;
        for (var type in el.events) removeEventListener(el, type);
        delete el.events;
        delete el.handle;
    }

    add(el, type, handler) {
        if (el.setInterval && (el != window && !el.frameElement))
            el = window;

        if (lx.isArray(handler)) {
            handler[1].context = handler[0];
            handler = handler[1];
        }

        if (!handler.guid) handler.guid = ++guid;
        this.appoint(el);
        if (el.events[type] === undefined) el.events[type] = {};

        if (el.events[type][handler.guid] === undefined) {
            addEventListener(el, type);
            el.events[type][handler.guid] = handler;
        }
    }

    remove(el, type, handler) {
        if (type == undefined && el.events) {
            for (var i in el.events) lx.app.domEvent.remove( el, i );
            return;
        }

        var handlers = el.events && el.events[type];
        if (!handlers) return;

        if (handler === undefined) {
            for ( var handle in handlers ) {
                delete el.events[type][handle];
            }
        } else {
            if (handler.guid in handlers)
                delete handlers[handler.guid].context;
            delete handlers[handler.guid];
        }
        for (var any in handlers) return;

        removeEventListener(el, type);
        delete el.events[type];
        for (var any in el.events) return;

        try {
            delete el.handle;
            delete el.events;
        } catch(e) { // IE
            el.removeAttribute('handle');
            el.removeAttribute('events');
        }
    }

    has(el, type, handler) {
        if (!el.events || !el.events[type]) return false;
        if (handler) return (el.events[type][handler.guid] === handler);
        return true;
    }
}

function fixEvent(e) {
    e = e || event;
    if (e.isFixed) return e;

    e.preDefault = e.preDefault || function(){this.returnValue = false};
    e.stopPropagation = e.stopPropagaton || function(){this.cancelBubble = true};

    if (!e.target) e.target = e.srcElement;
    if (!e.relatedTarget && e.fromElement)
        e.relatedTarget = e.fromElement == e.target ? e.toElement: e.fromElement;
    if (e.pageX == null && e.clientX != null) {
        var html = document.documentElement, body = document.body;
        e.pageX = e.clientX + (html && html.scrollLeft || body && body.scrollLeft || 0) - (html.clientLeft || 0);
        e.pageY = e.clientY + (html && html.scrollTop || body && body.scrollTop || 0) - (html.clientTop || 0);
    }
    if (!e.which && e.button)
        e.which = (e.button & 1 ? 1: ( e.button & 2 ? 3: ( e.button & 4 ? 2: 0 ) ));

    e.isFixed = true;
    return e;
}

function commonHandle(e) {
    var el = this.__lx ? this.__lx : this;

    if ( el && el.lxHasMethod('disabled') && el.disabled() ) {
        e.preventDefault();
        e.cancelBubble = true;
        e.stopPropagation();
        return;
    }

    e = fixEvent(e);
    var handlers = this.events[e.type];
    for (var g in handlers) {
        var handler = handlers[g],
            context = handler.context ? handler.context : (el ? el : this);
        if (lx.isInstance(el, lx.Rect) && e.targetWidget === undefined)
            e.targetWidget = el;
        if (handler.call(context, e) === false) {
            e.preventDefault();
            e.stopPropagation();
        }

        if (e.stopNow) break;
    }
}

function addEventListener(el, type) {
    if (el.addEventListener)
        el.addEventListener(type, el.handle, false);
    else if (el.attachEvent)
        el.attachEvent('on' + type, el.handle);
}

function removeEventListener(el, type) {
    if (el.removeEventListener)
        el.removeEventListener(type, el.handle, false);
    else if (el.detachEvent)
        el.detachEvent('on' + type, el.handle);
}
