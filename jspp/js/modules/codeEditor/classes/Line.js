#lx:public;

class Line {
    constructor(context, span) {
        if (span.span !== undefined) span = span.span;
        for (var pre=span; pre && pre.innerHTML!='<br>'; pre=pre.previousElementSibling) {}

        this.context = context;
        this.first = pre
            ? this.context.getSpanWrapper(pre.nextElementSibling)
            : this.context.getSpanWrapper(span.parentElement.childNodes[0]);
    }

    indent() {
        if ( this.first.text()[0] == ' ' || this.first.text()[0] == '\t' ) return this.first.text();
        else return '';
    }

    firstWordSpan() {
        if ( this.first.text()[0] == ' ' || this.first.text()[0] == '\t' ) return this.first.next();
        else return this.first;
    }

    next() {
        for (var next=this.first; next && next.text()!=String.fromCharCode(13); next=next.next()) {}
        if (!next) return null;
        for (var n=next; n && n.text()==String.fromCharCode(13); n=n.next()) {}
        if (!n) return null;
        return new Line(this.context, n);
    }

    commented() {
        if ( this.first.text() == '/'+'/' ) return true;
        if ( (this.first.text()[0] == ' ' || this.first.text()[0] == '\t') && this.first.next().text() == '/'+'/' ) return true;
        return false;
    }

    comment(indent) {
        var span;
        if (!indent || indent == this.indent().length) {
            var f = this.firstWordSpan();
            span = f.insertPre('/'+'/');
        } else {
            var spans = this.first.insertIn('/'+'/', indent);
            this.first = spans[0];
            span = spans[1];
        }
    }

    uncomment() {
        var first = this.firstWordSpan();
        var sp = first.next();
        if (sp.text() == ' ') sp.extract();
        else if (sp.len() > 1 && sp.text()[0] == ' ' && sp.text()[1] == '\t')
            sp.span.innerHTML = sp.text().substr(1);
        first.extract();
    }

    swapComment() {
        this.commented() ? this.uncomment() : this.comment();
    }
}
