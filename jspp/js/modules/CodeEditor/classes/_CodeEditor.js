#lx:public;

#lx:use lx.Textarea;


const _CodeEditor = {};
_CodeEditor.context = null;


_CodeEditor.langName = function() {
    if ( cr.context === null ) return '';
    return cr.context.editor.lang;
};

_CodeEditor.lang = function() {
    return cr.langs[ cr.langName() ];
};

_CodeEditor.style = function(type) {
    return cr.lang().styles[type];
};

_CodeEditor.charType = function(ch) {
    if ( ch == String.fromCharCode(13) ) return NaN;
    if ( ch.match(/[\wа-яё\d]/i) !== null ) return 1;//TODO TYPE 1
    if ( ch.match(/\s/) !== null ) return 2;//TODO TYPE 2
    return 3;//TODO TYPE 3
};

_CodeEditor.checkTextJoinValid = function(text0, text1) {
    if (text0 == String.fromCharCode(13) || text1 == String.fromCharCode(13) || text0 == '<br>' || text1 == '<br>') return false;
    var type0 = cr.charType(text0[0]),
        type1 = cr.charType(text1[0]);
    if (type0 != type1) return false;
    if (type0 == 3) return ( cr.lang().pareSymbols.indexOf( text0 + text1 ) != -1 );//TODO TYPE 3
    return true;
};

_CodeEditor.loading = function(mode) {
    if (!mode) {
        delete cr.loadingInfo;
        return;
    }

    cr.loadingInfo = {
        "'": false,
        '"': false,

        quoteOpened: function() { return (this['"'] || this['\'']); },
        commentCounter: 0,
        commentLine: false,
        commentInc: function() { this.commentCounter++; },
        commentDec: function() { if (this.commentCounter) this.commentCounter--; }
    }
};



_CodeEditor.COMMENT_OPEN = 0;
_CodeEditor.COMMENT_CLOSE = 1;
_CodeEditor.COMMENT_LINE = 2;


//======================================================================================================================
/* _CodeEditor.lang */

_CodeEditor.RESERVED = 0;
_CodeEditor.SPECIAL = 1;
_CodeEditor.SYMBOLS = 2;
_CodeEditor.FUNCTIONS = 3;
_CodeEditor.CONSTRUCTORS = 4;
_CodeEditor.VARIABLE = 5;
_CodeEditor.NUMERIC = 6;
_CodeEditor.CONSTANT = 7;
_CodeEditor.STRINGS = 8;
_CodeEditor.COMMENT = 9;

_CodeEditor.marker = {
    checkToColor: function(span) {
        var text = span.text();
        if (text.length == 6 && !isNaN(parseInt(text, 16))) {
            var pre = span.pre();
            if ( pre && pre.text() == '#' ) {
                span.span.style.backgroundColor = '#' + text;
                pre.span.style.backgroundColor = '#' + text;
            }
        }
    },

    styleType: function(span) {
        var text = span.text(),
            lang = cr.lang();

        if ( lx.isNumber(text) ) return cr.NUMERIC;
        if ( text == '.' && lx.isNumber(span.pre().text()) && lx.isNumber(span.next().text()) ) return cr.NUMERIC;
        if ( span.isCommentQuote() !== false ) return cr.COMMENT;
        if ( span.isQuote() ) return cr.STRINGS;
        if ( lang.symbols.indexOf(text) != -1 || lang.pareSymbols.indexOf(text) != -1 ) return cr.SYMBOLS;
        if ( lang.reserved.indexOf(text) != -1 ) return cr.RESERVED;
        if ( lang.special.indexOf(text) != -1 ) return cr.SPECIAL;

        // функции и конструкторы
        var next = span.next();
        if (next && next.text() == '(') {
            var pre = span.pre();
            if (pre && pre.text() == 'new') return cr.CONSTRUCTORS;
            return cr.FUNCTIONS;
        }

        var result = lang.check(span);

        return result;
    },

    actualizeStringStyles: function(first) {
        for (var next=first; next; next=next.next()) {
            var isQuote = next.isQuote(),
                preString = next.preString();
            if (!isQuote) next.string( preString );
            else if (isQuote) {
                if (!preString) {
                    delete next.lx().inString;
                    next.lx().open = true;
                } else if (isQuote == preString) {
                    delete next.lx().inString;
                    next.lx().open = false;
                } else {
                    delete next.lx().open;
                    next.lx().inString = preString;
                }
            }
        }
    },

    checkString: function(span) {
        var isQuote = span.isQuote(),
            preString = span.preString();

        // имеем дело не с кавычкой - просто проверяем не в строку ли пишется новый узел
        if (!isQuote) {
            if (preString) span.string(preString);
            return;
        }

        // если кавычка оказалась внутри другой кавычки
        if (preString && preString != isQuote) {
            span.lx().inString = preString;
            return;
        }

        if (!preString) {
            span.lx().open = true;
            // автодобавление второй кавычки
            if (cr.context.isUsingAutoActions()) {
                var next = span.next();
                if (next && (!next.len() || next.type() == 2)) {//TODO TYPE 2
                    span.insertNext(isQuote);
                    return;
                }
            }
        } else {  // (preString == isQuote)
            span.lx().open = false;
            // чтобы при автодобавлении предотвратить цикл актуализации
            if (cr.context.isUsingAutoActions() && span.pre().isQuote() == isQuote) return;
        }

        this.actualizeStringStyles( span.next() );
    },

    commentSpanDeleted: function(quote, next) {
        if (quote == cr.COMMENT_LINE)
            for (var n=next; n.len(); n=n.next()) n.uncomment(true);
        else if (quote == cr.COMMENT_OPEN)
            for (var n=next; n && !n.isCommentQuote(cr.COMMENT_CLOSE); n=n.next()) n.uncomment();
        else if (quote == cr.COMMENT_CLOSE) {
            var pre = next.pre();
            if (pre && pre.isCommentQuote() === false && pre.lx().commentCounter)
                for (var n=next; n && !n.isCommentQuote(cr.COMMENT_CLOSE); n=n.next()) n.comment();
        }
    },

    checkComment: function(span) {
        var isComment = span.isCommentQuote();

        // если имеем дело со знаком комментирования
        if (isComment !== false) {
            if (isComment == cr.COMMENT_LINE)
                for (var next=span.next(); next.len(); next=next.next()) next.comment(true);
            else if (isComment == cr.COMMENT_OPEN)
                for (var next=span.next(); next && !next.isCommentQuote(cr.COMMENT_CLOSE); next=next.next()) next.comment();
            else if (isComment == cr.COMMENT_CLOSE) {
                var pre = span.pre();
                if (pre && pre.isCommentQuote() === false && pre.lx().commentCounter)
                    for (var next=span.next(); next && !next.isCommentQuote(cr.COMMENT_CLOSE); next=next.next()) next.uncomment();
            }
            delete span.lx().commentCounter;
            delete span.lx().commentLine;
            // остальное надо проверять - не в комментарий ли пишется новый узел
        } else {
            var comm = 0, line = false;
            for (var pre=span.pre(); pre; pre=pre.pre()) {
                if ( pre.isCommentQuote(cr.COMMENT_CLOSE) ) comm--;
                else if ( pre.isCommentQuote(cr.COMMENT_OPEN) ) comm++;
                else if ( pre.isCommentQuote(cr.COMMENT_LINE) ) line = true;
                else {
                    if (pre.lx().commentCounter) comm += pre.lx().commentCounter;
                    if (pre.len() && pre.lx().commentLine) line = true;
                    break;
                }
            }
            if (comm < 0) comm = 0;
            if (comm || line) {
                span.addStyle( cr.style(cr.COMMENT) );
                span.lx().commentCounter = comm;
                span.lx().commentLine = line;
            }
        }
    },

    // автодополнение скобок
    autoBracket: function(span) {
        if ( span.text() == '(' ) { span.insertNext(')'); return true; }
        if ( span.text() == '[' ) { span.insertNext(']'); return true; }
        if ( span.text() == '{' ) { span.insertNext('}'); return true; }
        return false;
    },

    // сохранение табуляции при добавлении новой строки
    autoSpace: function(span) {
        if ( span.len() ) return false;

        var pre = span.pre();
        if (!pre) return false;

        var line = cr.context.getLine(pre),
            indent = line.indent();

        // если открыта скобка
        if (pre.text() == '{' || pre.text() == '(') {
            // особый случай - операторные скобки
            if (pre.text() == '{' && span.next().text() == '}') {
                if (indent !== '') span.insertNext(indent);
                span.insertNext(String.fromCharCode(13));
            }
            indent += '\t';
            // иначе надо проверить первое слово
        } else {
            var word = line.firstWordSpan().text(),
                arr = ['if', 'for', 'while'];
            if (arr.indexOf(word) != -1) indent += '\t';
        }

        if (indent !== '') {
            var next = span.insertNext(indent);
            cr.range().setCaret( next, next.len() );
        }
        return true;
    },

    autoHint: function(span) {
        if (cr.context.hint.isActive() || cr.context.hint.isSwitchedOff()) {
            return false;
        }
        if (span.type() != 1) return false;//TODO TYPE 1
        if (span.len() < cr.context.hint.getMinLen()) return false;

        var slash = String.fromCharCode(92);
        var text = span.text(),
            cod = cr.context.editor.getText(),
            re = new RegExp(slash+'b'+text+'['+slash+'w'+slash+'d]*?'+slash+'b', 'g'),
            boof = cod.match(re),
            arr = [],
            empty = true;

        if (!boof) return false;
        for (var i=0, l=boof.length; i<l; i++) {
            if ( boof[i] == text ) continue;
            arr[ boof[i] ] = 1;
            empty = false;
        }
        if (empty) return false;

        var s = span.span,
            l = s.offsetLeft,
            t = s.offsetTop + s.offsetHeight;
        boof = [];
        for (var i in arr) boof.push(i);
        cr.context.hint.show(l, t, boof);

        return true;
    },

    autoActions: function(span) {
        if (this.autoBracket(span)) return;
        if (this.autoSpace(span)) return;

        if (!this.autoHint(span)) cr.context.hint.hide();
    },

    check: function(span) {
        span.resetStyle();

        var st = this.styleType( span );
        if (st != -1) span.addStyle( cr.style(st) );

        this.checkString(span);
        this.checkComment(span);

        this.checkToColor(span);

        // некоторые автоматизмы
        if (cr.context.isUsingAutoActions()) this.autoActions(span);
    },

    loadingCheck: function(span) {
        var st = this.styleType( span );
        if (st != -1) span.addStyle( cr.style(st) );

        var isCommentQuote = span.isCommentQuote();
        if (isCommentQuote === cr.COMMENT_LINE) cr.loadingInfo.commentLine = true;
        else if (isCommentQuote === cr.COMMENT_OPEN) cr.loadingInfo.commentInc();
        else if (isCommentQuote === cr.COMMENT_CLOSE) cr.loadingInfo.commentDec();
        else if ( cr.loadingInfo.commentCounter || cr.loadingInfo.commentLine ) {
            span.addStyle( cr.style(cr.COMMENT) );
            span.lx().commentCounter = cr.loadingInfo.commentCounter;
            span.lx().commentLine = cr.loadingInfo.commentLine;
        }
        if (!span.len()) cr.loadingInfo.commentLine = false;

        var opened = span.preString(),
            isQuote = span.isQuote();
        if (isQuote && (!opened || opened == isQuote)) {
            cr.loadingInfo[isQuote] = !cr.loadingInfo[isQuote];
            span.lx().open = cr.loadingInfo[isQuote];
        } else if (cr.loadingInfo.quoteOpened()) {
            span.string(opened);
        }
    }
};



_CodeEditor.langs = {
    php: {
        type: 'php',

        styles: [
            'lxCR-php-rsv', /* RESERVED */
            'lxCR-php-spc', /* SPECIAL */
            'lxCR-php-smb', /* SYMBOLS */
            'lxCR-php-fnc', /* FUNCTIONS */
            'lxCR-php-cns', /* CONSTRUCTORS */
            'lxCR-php-var', /* VARIABLE */
            'lxCR-php-nmr', /* NUMERIC */
            'lxCR-php-cst', /* CONSTANT */
            'lxCR-php-str', /* STRINGS */
            'lxCR-php-cmt'  /* COMMENT */
        ],

        reserved: [
            'require_once', 'return', 'if', 'else', 'elseif', 'new', 'class',
            'const', 'extends', 'private', 'public', 'protected',
            'for', 'foreach', 'continue', 'do', 'while'
        ],

        special: [],

        symbols: [ '+', '*', '-', '/', '!', '@', '.', '&lt;', '&gt;', '=' ],
        pareSymbols: [ '/'+'*', '*'+'/', '/'+'/', '===', '==', '||', '&&', '&amp;&amp;', '++', '--', '::' ],

        strings: [ '"', '\'' ],

        commentQuotes: [
            /*open*/ '/'+'*',
            /*close*/ '*'+'/',
            /*line*/ '/'+'/'
        ],

        check: function(span) {
            // переменные, начинающиеся с $
            if (span.text() == '$') return cr.VARIABLE;

            var pre = span.pre();
            if (pre && pre.text() == '$' && span.type() == 1) return cr.VARIABLE;//TODO TYPE 1
            // все для них же - если пропала связь с $
            if (pre && pre.text() == '$' && span.type() != 1) cr.marker.check(span.next());//TODO TYPE 1
            // объявление класса
            if (pre && pre.pre() && pre.pre().text() == 'class') return cr.FUNCTIONS;

            var next = span.next();
            // Классы, у которых вызвана статика
            if (next && next.text() == '::') return cr.FUNCTIONS;

            // все что не переменная в php - константа
            if (span.type() == 1) return cr.CONSTANT;//TODO TYPE 1
            return -1;
        }
    },

    js: {
        type: 'js',

        styles: [
            'lxCR-js-rsv', /* RESERVED */
            'lxCR-js-spc', /* SPECIAL */
            'lxCR-js-smb', /* SYMBOLS */
            'lxCR-js-fnc', /* FUNCTIONS */
            'lxCR-js-cns', /* CONSTRUCTORS */
            'lxCR-js-var', /* VARIABLE */
            'lxCR-js-nmr', /* NUMERIC */
            'lxCR-js-cst', /* CONSTANT */
            'lxCR-js-str', /* STRINGS */
            'lxCR-js-cmt'  /* COMMENT */
        ],

        reserved: ['new', 'return', 'if', 'else', 'class', 'extends', 'var', 'let', 'const'],

        special: ['lx', 'this'],

        symbols: [ '+', '*', '-', '/', '!', '&lt;', '&gt;', '=' ],
        pareSymbols: [ '/'+'*', '*'+'/', '/'+'/', '===', '==', '=&lt', '||', '&&', '&amp;&amp;', '++', '--' ],

        strings: [ '"', '\'' ],

        commentQuotes: [
            /*open*/ '/'+'*',
            /*close*/ '*'+'/',
            /*line*/ '/'+'/'
        ],

        check: function(span) {
            return -1;
        }
    }
};
/* _CodeEditor.lang */
//======================================================================================================================

var cr = _CodeEditor;
