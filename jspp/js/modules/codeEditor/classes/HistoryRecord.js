#lx:public;

class HistoryRecord {
    constructor(context, type, info, name) {
        this.context = context;
        this.spanId = name || context.history.genId();
        this.type = type;
        this.info = info;
    }

    span() {
        return this.context.getSpanWrapper(document.getElementsByName(this.spanId)[0]);
    }

    inputBack() {
        var s = this.span();
        if (this.info.pos == 0 && this.info.amt == s.len()) {
            s.extract();
            return;
        }
        var text = s.text();
        text = text.substr(0, this.info.pos) + text.substr(this.info.pos + this.info.amt);
        s.text(text);
        this.context.getRange().setCaret(s, this.info.pos);
    }

    deleteBack() {
        var s = this.span(),
            offset = (this.info.way==0) ? this.info.pos : this.info.pos - this.info.text.length;
        var text = s.text();
        text = text.substr(0, offset) + this.info.text + text.substr(offset);
        s.text(text);
        this.context.getRange().setCaret(s, this.info.pos);
    }

    extractBack() {
        var s = this.span(),
            span;
        if (this.info.offset !== undefined) {
            var spans = s.split(this.info.offset);
            if (this.info.nextName) spans[1].span.setAttribute('name', this.info.nextName);
            span = spans[0].insertNext(this.info.text);
            if (this.info.name) span.span.setAttribute('name', this.info.name);
        } else {
            span = s.insertPre(this.info.text);
            if (this.info.name) span.span.setAttribute('name', this.info.name);
        }
        this.context.getRange().setCaret(span, span.len());
    }

    pasteBack() {
        if (this.spanId == this.info.pre) {
            var s = this.span(),
                text = s.text();
            s.text( text.substr(0, this.info.preOffset) + text.substr(this.info.nextOffset) );
            this.context.getRange().setCaret(s, this.info.preOffset);
            return;
        }

        var next = this.span(),
            pre = (this.info.pre) ? this.context.getSpanWrapper(document.getElementsByName(this.info.pre)[0]) : null,
            arr = [],
            offset = 0;
        for (var p = next.pre(); p && !p.equal(pre); p=p.pre()) arr.push(p);
        for (var i=0, l=arr.length; i<l; i++) arr[i].del();

        if (this.info.nextOffset)
            next.text( next.text().substr(this.info.nextOffset) );
        if (pre) {
            next = pre;
            offset = this.info.preOffset;
            if (this.info.preOffset < pre.len()) pre.text( pre.text().substr(0, this.info.preOffset) );
            pre.joinNext();
        }
        this.context.getRange().setCaret(next, offset);
    }

    massdelBack() {
        var s = this.span();
        var next, pre,
            nextOffset = 0, preOffset;
        if (this.info.names === undefined) {
            pre = s;
            next = s;
            preOffset = this.info.offset;
            nextOffset = preOffset + this.info.text.length;
            s.text( s.text().substr(0, this.info.offset) + this.info.text + s.text().substr(this.info.offset) );
        } else {
            next = s;
            if ( this.info.preLen !== undefined ) {
                var spans = s.split( this.info.preLen );
                pre = spans[0];
                next = spans[1];
                if ( this.info.nextName !== null ) next.span.setAttribute('name', this.info.nextName);
            } else pre = next.pre();

            if (pre) {
                preOffset = pre.len();
                if (this.info.preText !== undefined) pre.text( pre.text() + this.info.preText );
            } else {
                pre = this.context.editor.getElement().childNodes[0];
                preOffset = 0;
            }
            if (this.info.nextText !== undefined) {
                nextOffset = this.info.nextText.length;
                next.text( this.info.nextText + next.text() );
            }
            var boof = document.createElement('span');
            boof.innerHTML = this.context.editor.markText(this.info.text);
            for (var i=0, l=boof.childNodes.length; i<l; i++) {
                var span = next.insertPre( boof.childNodes[i].innerHTML );
                if (this.info.names[i] !== null) span.span.setAttribute('name', this.info.names[i]);
            }
        }
        if (preOffset == pre.len()) { pre = pre.next(); preOffset = 0; }
        if (this.info.seq) this.context.getRange().setCaret(pre, preOffset, next, nextOffset);
        else this.context.getRange().setCaret(next, nextOffset, pre, preOffset);  // это так не работает
    }

    restore() {
        this.context.history.off();
        this.context.trigger(EventType.AUTO_ACTION_START_FREE_OPERATION);
        switch (this.type) {
            case History.EVENT_INPUT: this.inputBack(); break;
            case History.EVENT_DELETE: this.deleteBack(); break;
            case History.EVENT_EXTRACT: this.extractBack(); break;
            case History.EVENT_PASTE: this.pasteBack(); break;
            case History.EVENT_MASS_DELETE: this.massdelBack(); break;
        }
        this.context.trigger(EventType.AUTO_ACTION_FINISH_FREE_OPERATION);
        this.context.history.on();
    }
}
