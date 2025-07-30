#lx:public;

class SelectionRange {
    constructor(context) {
        this.context = context;
        this.selection = null;
        this.anchor = null;
        this.focus = null;
        this.anchorOffset = null;
        this.focusOffset = null;
        this.reset();
    }

    reset() {
        var sel = document.getSelection();
        this.selection = sel;
        if (sel.anchorNode === null) return;
        // проверка для бр-спанов
        var s0 = sel.anchorNode,
            s1 = sel.focusNode;
        if ( s0.parentElement !== this.context.editor.getElement() ) s0 = s0.parentElement;
        if ( s1.parentElement !== this.context.editor.getElement() ) s1 = s1.parentElement;
        this.anchor = this.context.getSpanWrapper(s0);
        this.focus = this.context.getSpanWrapper(s1);
        this.anchorOffset = sel.anchorOffset;
        this.focusOffset = sel.focusOffset;
    }

    isCaret() {
        return ( this.anchor.span === this.focus.span && this.anchorOffset == this.focusOffset );
    }

    rightSequens() {
        if (this.anchor.span === this.focus.span) return (this.anchorOffset <= this.focusOffset);
        if ( this.anchor.span.offsetTop < this.focus.span.offsetTop
            || (this.anchor.span.offsetTop == this.focus.span.offsetTop && this.anchor.span.offsetLeft < this.focus.span.offsetLeft) )
            return true;
        return false;
    }

    edges() {
        if ( this.rightSequens() ) return [ this.anchor, this.focus, this.anchorOffset, this.focusOffset ];
        return [ this.focus, this.anchor, this.focusOffset, this.anchorOffset ];
    }

    allSpans() {
        if (this.anchor.span === this.focus.span) return [this.anchor];
        var edges = this.edges(),
            result = [];
        for (var temp=edges[0]; temp.span!==edges[1].span; temp=temp.next())
            result.push( temp );
        result.push( edges[1] );
        return result;
    }

    lines() {
        if (this.anchor.span === this.focus.span)
            return this.context.getLinesList([this.context.getLine(this.anchor)]);
        var edges = this.edges(),
            result = [this.context.getLine(edges[0])];
        for (var temp=edges[0]; temp.span!==edges[1].span; temp=temp.next())
            if (temp.text() == String.fromCharCode(13) && temp.next().text() != String.fromCharCode(13))
                result.push(this.context.getLine(temp.next()));
        return this.context.getLinesList(result);
    }

    caretOnStart() {
        return ((this.anchorOffset == 0 && this.anchor.pre() == null)
            || (this.focusOffset == 0 && this.focus.pre() == null));
    }

    caretOnEnd() {
        function end(s, offset) { return (offset == s.len() && (s.next() === null || s.next().next() === null)); };
        return ( end(this.anchor, this.anchorOffset) || end(this.focus, this.focusOffset) );
    }

    setCaret(span, offset, span1, offset1) {
        if (offset > span.len()) {
            console.log('LX: offset('+offset+') is larger then span.length('+span.len()+')');
            offset = span.len();
        }

        var r = document.createRange();
        r.setStart(span.span.childNodes[0], offset);

        if (span1 === undefined) r.collapse();
        else r.setEnd(span1.span.childNodes[0], offset1);

        this.selection.removeAllRanges();
        this.selection.addRange(r);
    }

    delRange() {
        var edges = this.edges();

        // сначала самое простое - все в перделах одного спана
        if ( edges[0].equal(edges[1]) ) {
            // вышли на удаление спана
            if (edges[2] == 0 && edges[3] == edges[0].len()) {
                edges[0].extract();
                return;
            }
            // просто текст из середины спана вырезан
            text = edges[0].text().substr( edges[2], edges[3] - edges[2] );
            edges[0].text( edges[0].text().substr(0, edges[2]) + edges[0].text().substr(edges[3]) );
            this.context.trigger(EventType.HISTORY_MASS_DELETE, {
                span: edges[0],
                info: {
                    text: text,
                    offset: edges[2],
                    seq: this.rightSequens()
                }
            });
            this.setCaret(edges[0], edges[2]);
            return;
        }

        var spans = this.allSpans(),
            pre = edges[0],
            next = edges[1],
            start = 1,
            finish = spans.length-1,
            text = '',
            names = [],
            sequens = this.rightSequens(),
            offset = 0;

        // если первый спан с нулевой позиции - он будет удален
        if ( edges[2] == 0 ) {
            start = 0;
            pre = pre.pre();
            if (pre) edges[2] = pre.len();
        }

        // если последний спан с конечной позиции, он будет удален
        if ( edges[3] == next.len() ) {
            next = next.next();
            edges[3] = 0;
            finish++;
        }

        // удалим все промежуточные спаны
        this.context.trigger(EventType.HISTORY_START_FREE_OPERATION);
        for (var i=start; i<finish; i++) {
            text += spans[i].text();
            names.push(spans[i].name());
            spans[i].del();
        }
        text = text.replace(/&lt;/g, '<');
        text = text.replace(/&gt;/g, '>');
        text = text.replace(/&amp;/g, '&');

        var info = { text: text, names: names, seq: sequens };
        if (next.len()) {
            info.nextText = next.text().substr( 0, edges[3] );
            next.span.innerHTML = next.span.innerHTML.substr(edges[3]);
        }
        if (pre) {
            if (pre.len()) {
                info.preText = pre.text().substr( edges[2] );
                pre.span.innerHTML = pre.span.innerHTML.substr(0, edges[2]);
            }
            var name = next.name(),
                len = pre.len();
            if (pre.joinNext()) {
                info.nextName = name;
                info.preLen = len;
                next = pre;
                offset = len;
            }
        }
        this.context.trigger(EventType.HISTORY_FINISH_FREE_OPERATION);
        this.context.trigger(EventType.HISTORY_MASS_DELETE, {
            span: next,
            info
        });
        this.setCaret(next, offset);
    }

    del(key) {
        // ключ нужен только для каретки, range удаляется одинаково и 8 и 46
        if (this.isCaret()) {
            if (key == 8 && this.caretOnStart() ) return;
            if (key == 46 && this.caretOnEnd() ) return;
            this.anchor.delChar( this.anchorOffset, (key==8)?-1:0 );
        } else this.delRange();
    }

    addChar(key) {
        if (this.isCaret()) {
            this.anchor.addChar(key, this.anchorOffset);
        } else {
            this.del();
            this.reset();
            this.addChar(key);
        }
    }

    pasteText(text) {
        this.context.trigger(EventType.AUTO_ACTION_START_FREE_OPERATION);
        if (this.isCaret()) {
            var next;

            if (this.anchorOffset == 0)
                next = this.anchor;
            else if (this.anchorOffset == this.anchor.len())
                next = this.anchor.next();
            else {
                this.setCaret(this.anchor.split(this.anchorOffset)[1], 0);
                this.reset();
                this.pasteText(text);
                return;
            }

            text = this.context.editor.markText(text);
            var boof = document.createElement('span');
            boof.innerHTML = text;

            // чтобы поспаново в историю не писалось
            this.context.trigger(EventType.HISTORY_START_FREE_OPERATION);
            // для истории
            var info = {
                pre: null,
                preOffset: 0,
                nextOffset: 0
            };

            // первый узел идет на попытку слияния с имеющимся в тексте
            var temp = next.insertPre( boof.childNodes[0].innerHTML ),
                pre = temp.pre();
            if (pre) {
                info.preOffset = pre.len();
                if (pre.joinNext()) temp = pre;
                info.pre = pre;
            }

            // вставляем остальные узлы
            for (var i=1, l=boof.childNodes.length; i<l; i++)
                temp = next.insertPre( boof.childNodes[i].innerHTML );

            // последний узел идет на попытку слияния с имеющимся в тексте
            var offset = temp.len();
            if (temp.joinNext()) {
                info.nextOffset = offset;
                next = temp;
            }
            // каретка ставится в конец вставленного фрагмента
            this.setCaret(temp, offset);

            // включить историю обратно
            this.context.trigger(EventType.HISTORY_FINISH_FREE_OPERATION);
            this.context.trigger(EventType.HISTORY_PASTE, {
                span: next,
                info
            });
        } else {
            this.del();
            this.reset();
            this.pasteText(text);
        }
        this.context.trigger(EventType.AUTO_ACTION_FINISH_FREE_OPERATION);
    }

    paste() {
        var _t = this,
            span = _t.anchor,
            offset = _t.anchorOffset;
        _t.selection.removeAllRanges();

        (new lx.Textarea())
            .opacity(0)
            .focus()
            .addEventListener('keyup', function() {
                _t.setCaret(span, offset);
                var text = this.value();
                this.del();
                _t.pasteText(text);
            });
    }
}
