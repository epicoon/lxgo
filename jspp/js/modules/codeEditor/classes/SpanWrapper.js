#lx:public;

class SpanWrapper {
    constructor(context, span) {
        this.context = context;
        this.span = span;
    }

    lx() {
        if (this.span.lx === undefined) this.span.lx = {};
        return this.span.lx;
    }

    text(t) {
        if (t !== undefined) {
            this.span.innerHTML = t;
            this.context.getHighlighter().check(this);
            return;
        }
        if (this.span.innerHTML == '<br>') return String.fromCharCode(13);
        return this.span.innerHTML;
    }

    len() {
        if (this.text() == String.fromCharCode(13)) return 0;
        if (this.text() == '&lt;') return 1;
        if (this.text() == '&gt;') return 1;
        if (this.text() == '&amp;') return 1;
        if (this.text() == '&amp;&amp;') return 2;
        return this.text().length;
    }

    name() {
        return this.span.getAttribute('name');
    }

    equal(s) {
        if (s === null) return false;
        return (this.span === s.span);
    }

    checkCaret(offset) {
        var r = this.context.getRange();
        if (!r.isCaret()) return false;

        // если оффсет не передан, вернет оффсет, если он есть на этом спане, иначе false
        if (offset === undefined) {
            if (this.equal(r.anchor)) return r.anchorOffset;
            var pre = this.pre();
            if (!pre) return false;
            if (pre.equal(r.anchor) && pre.len() == r.anchorOffset) return 0;
        }

        // если оффсет передан, проверит именно эту позицию
        if (!offset) {
            var pre = this.pre();
            if (!pre) return false;
            if (pre.equal(r.anchor) && pre.len() == r.anchorOffset) return true;
        }
        if (!this.equal(r.anchor)) return false;
        if (offset == r.anchorOffset) return true;
        return false;
    }

    setText(text, shift) {  // сам следит за кареткой
        if (shift === undefined) shift = 0;
        var offset = this.checkCaret();
        this.text(text);
        if (offset !== false) this.context.getRange().setCaret(this, offset + shift);
    }

    checkJoin(span) {
        span = span || this.next();
        if (!span) return false;
        return cr.checkTextJoinValid( this.text(), span.text() );
    }

    joinNext() {
        if ( !this.checkJoin() ) return false;
        var next = this.next();
        this.setText( this.text() + next.text() );
        next.del();
        return this;
    }

    split(pos) {  // как инструмент вне истории
        this.context.trigger(EventType.HISTORY_START_FREE_OPERATION);
        var span = this.insertNext( this.span.innerHTML.substr(pos) );
        this.context.trigger(EventType.HISTORY_FINISH_FREE_OPERATION);

        this.span.innerHTML = this.span.innerHTML.substr(0, pos);
        return [this, span];
    }

    del() {  // скидывает каретку на предыдущий спан, если нужно
        this.context.hint.hide();

        var isQuote = this.isQuote(),
            isCommentQuote = this.isCommentQuote();

        var next = this.next(),
            r = this.context.getRange(), node;
        if (this.equal(r.anchor)) node = this.next();
        this.span.parentElement.removeChild(this.span);
        if (node) r.setCaret(node, 0);

        if (isQuote) this.context.getHighlighter().actualizeStringStyles(next);
        else if (isCommentQuote !== false) this.context.getHighlighter().commentSpanDeleted(isCommentQuote, next);
    }

    extract() {  // отличается от del() - не просто удаляет, но и объединит оказавшиеся соседями спаны, если это возможно
        var info = {
            text: this.text(),
            name: this.span.getAttribute('name'),
            nextName: this.next().span.getAttribute('name')
        };

        var pre = this.pre(),
            next = this.next();
        this.del();
        if (pre) {
            var len = pre.len();
            if (pre.joinNext()) info.offset = len;
        }

        if (info.offset === undefined) pre = next;

        this.context.trigger(EventType.HISTORY_EXTRACT, {
            span: pre,
            info
        });
    }

    clearColor() {
        if (this.span.style.backgroundColor == '') return;
        this.span.style.backgroundColor = '';
        var pre = this.pre();
        if (pre) pre.span.style.backgroundColor = '';
    }

    resetStyle() {
        var str = this.span.classList.contains( cr.style(cr.STRINGS) ),
            com = this.span.classList.contains( cr.style(cr.COMMENT) );
        this.span.className = '';
        this.clearColor();
        if (str) this.span.classList.add( cr.style(cr.STRINGS) );
        if (com) this.span.classList.add( cr.style(cr.COMMENT) );
    }

    addStyle(st) {
        this.span.classList.add(st);
    }

    delStyle(st) {
        this.span.classList.remove(st);
    }

    toggleStyle(st, bool) {
        this.span.classList.toggle(st, bool);
    }

    hasStyle(st) {
        return this.span.classList.contains(st);
    }

    comment(bool) {
        if ( this.isCommentQuote() !== false ) return;
        if (!this.span.lx) this.span.lx = { commentCounter: 0, commentLine: false };
        if (bool === undefined) this.span.lx.commentCounter++;
        else this.span.lx.commentLine = true;
        this.addStyle( cr.style(cr.COMMENT) );
    }

    uncomment(bool) {
        if (!this.span.lx || (!this.span.lx.commentCounter && !this.span.lx.commentLine)) return;
        if (bool === undefined) this.span.lx.commentCounter--;
        else this.span.lx.commentLine = false;
        if (!this.span.lx.commentCounter && !this.span.lx.commentLine)
            this.delStyle( cr.style(cr.COMMENT) );
    }

    string(bool) {
        this.lx().inString = bool;
        this.toggleStyle( cr.style(cr.STRINGS), bool );
    }

    next() {
        var next = this.span.nextElementSibling;
        if (!next) return null;
        return this.context.getSpanWrapper(next);
    }

    pre() {
        var pre = this.span.previousElementSibling;
        if (!pre) return null;
        return this.context.getSpanWrapper(pre);
    }

    insertPre(key) {
        var span = this.context.createSpan(key);
        this.span.parentElement.insertBefore( span.span, this.span );
        this.context.getHighlighter().check(span);
        if ( span.len() && this.checkCaret(0) )  // не нужно менять каретку если был добавлен переход на новую строку
            this.context.getRange().setCaret( span, span.len() );
        return span;
    }

    insertNext(key) {
        var span = this.context.createSpan(key),
            next = this.next();
        if (next === null) this.span.parentElement.appendChild(span.span);
        else this.span.parentElement.insertBefore(span.span, next.span);
        this.context.getHighlighter().check(span);
        if ( this.checkCaret(this.len()) ) {
            if (!span.len()) this.context.getRange().setCaret( span.next(), 0 );  // опять из-за энтера замут
            else this.context.getRange().setCaret( span, span.len() );
        }
        return span;
    }

    insertIn(key, offset) {
        var caret = this.checkCaret();
        var spans = this.split(offset),
            spliter = spans[1].insertPre(key);
        this.context.getHighlighter().check( spans[0] );
        this.context.getHighlighter().check( spans[1] );
        if (caret) this.context.getRange().setCaret(spliter, 1);
        return [ spans[0], spliter, spans[1] ];
    }

    EOL() {
        return (this.span.innerHTML == '<br>');
    }

    isCommentQuote(type) {
        var index = cr.lang().commentQuotes.indexOf(this.text());
        if (index == -1) return false;
        if (type === undefined) return index;
        return (index == type);
    }

    isQuote() {
        var index = cr.lang().strings.indexOf(this.text());
        if (index == -1) return false;
        return cr.lang().strings[index];
    }

    type() {
        if (this.span.innerHTML == '<br>') return 2;//TODO TYPE 2
        return cr.charType(this.span.innerHTML[0]);
    }

    preText(text) {
        for (var pre=this.pre(); pre && pre.text()!=text; pre=pre.pre()) {}
        return pre;
    }

    inString() {
        return this.span.classList.contains( cr.style(cr.STRINGS) );
    }

    preString() {
        var pre = this.pre();
        if (!pre) return false;
        if (pre.lx().inString) return pre.lx().inString;
        if (pre.lx().open) return pre.isQuote();
        return false;
    }

    nextWordSpan() {
        for (var temp=this.next(); temp && temp.type()==2; temp=temp.next()) {}//TODO TYPE 2
        return temp;
    }

    preWordSpan() {
        for (var pre=this.pre(); pre && pre.type()==2; pre=pre.pre()) {}//TODO TYPE 2
        return pre;
    }

    addChar(key, offset) {
        // типы совпадают
        if (cr.checkTextJoinValid(this.text(), key) || cr.checkTextJoinValid(key, this.text())) {
            this.context.trigger(EventType.HISTORY_INPUT, {
                span: this,
                info: {pos: offset, amt: key.length}
            });

            var text = this.text();
            this.setText( text.substr(0, offset) + key + text.substr(offset), 1 );
            return;
        }

        // offset == 0 только в начале строки, слить символ в предыдущий спан не выйдет, т.к. это бр-спан. Остается только добавлять новый
        if (offset == 0) this.insertPre(key);
        // добавляем символ в конец - или слить в следующий спан, или добавить новый спан после текущего
        else if (offset == this.len()) {
            var next = this.next();
            if ( next && cr.checkTextJoinValid(key, next.text()) ) next.addChar(key, 0);
            else this.insertNext(key);
            // разбиваем текущий спан новым
        } else this.insertIn(key, offset);
    }

    delChar(offset, shift) {
        if (shift === undefined) shift = 0;
        offset += shift;
        var l = this.len();
        if (offset > l) return false;

        // наводка на предыдущий спан
        if (offset == -1) {
            var pre = this.pre(),
                len = pre.len();
            if (len) pre.delChar(len - 1);
            else pre.extract(); // \r
            return;
        }

        // наводка на следующий спан
        if (offset == l) {
            var next = this.next(),
                len = next.len();
            if (len) next.delChar(0);
            else next.extract();  // \r
            return;
        }

        // длина говорит о том, что спан будет удален
        if (l < 2) {
            this.extract();
            return;
        }

        // собственно удаление символа
        this.context.trigger(EventType.HISTORY_DELETE, {
            span: this,
            info: {
                text: this.text()[offset],
                pos: offset - shift,
                way: shift
            }
        });
        var isCommentQuote = this.isCommentQuote();
        this.setText(this.span.innerHTML.substr(0, offset) + this.span.innerHTML.substr(offset+1), shift);
        if (isCommentQuote !== false) this.context.getHighlighter().commentSpanDeleted(isCommentQuote, this.next());
    }
}
