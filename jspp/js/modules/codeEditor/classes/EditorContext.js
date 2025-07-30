#lx:public;

class EditorContext {
    constructor(editor) {
        this._editor = editor;
        this._events = new lx.EventDispatcher();
        this._range = new SelectionRange(this);
        this._hint = new HintHolder(this)
        this._history = new History(this);

        this._useAutoCurrent = true;
        this._useAutoSetting = true;
        
        this._events.subscribe(
            EventType.AUTO_ACTION_START_FREE_OPERATION,
            ()=>this._useAutoCurrent = false
        );
        this._events.subscribe(
            EventType.AUTO_ACTION_FINISH_FREE_OPERATION,
            ()=>this._useAutoCurrent = this._useAutoSetting
        );
    }

    get editor() { return this._editor; }
    get hint() { return this._hint; }
    get history() { return this._history; }
    get events() { return this._events; }

    isUsingAutoActions() {
        return this._useAutoCurrent;
    }
    
    getHighlighter() {
        return cr.marker;
    }

    getRange() {
        this._range.reset();
        return this._range;
    }

    getSpanWrapper(span) {
        return new SpanWrapper(this, span);
    }
    
    trigger(eventName, data) {
        this.events.trigger(eventName, data);
    }

    createSpan(key) {
        var span = document.createElement('span');
        span.innerHTML = (key == String.fromCharCode(13)) ? '<br>' : key;
        span = this.getSpanWrapper(span);

        this.trigger(EventType.HISTORY_INPUT, {
            span,
            info: {pos: 0, amt: key.length}
        });

        return span;
    }
    
    getLine(span) {
        return new Line(this, span);
    }
    
    getLinesList(list) {
        return new LinesList(list);
    }
}
