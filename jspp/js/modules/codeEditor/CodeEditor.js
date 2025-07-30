// @lx:module lx.CodeEditor;

// @lx:use lx.Table;

// @lx:require -R classes/;

/*
* TODO
* 
* В процессе рефакторинга
* файл ./classes/_CodeEditor.js разнёс на разные классы, но не закончил
* */

// @lx:namespace lx;
class CodeEditor extends lx.Element {
    static initCss(css) {
        css.addClass('lxCR-back', {
            fontFamily: 'Courier',
            tabSize: 4,
            backgroundColor: '#272822',
            color: '#F8F8F2'
        });
        css.addClass('lxCR-hint', {
            backgroundColor: '#66FF99 !important'
        });

        // JavaScript
        // -- reserved
        css.addClass('lxCR-js-rsv', {
            fontWeight: 'bold',
            color: '#DE3453'
        });
        // -- symbols
        css.addClass('lxCR-js-smb', {
            color: '#DE3453'
        });
        // -- special
        css.addClass('lxCR-js-spc', {
            fontStyle: 'italic',
            color: '#AAAAFF'
        });
        // -- function
        css.addClass('lxCR-js-fnc', {
            fontStyle: 'italic',
            color: '#80FFFF'
        });
        // -- numbers
        css.addClass('lxCR-js-nmr', {
            color: '#DE91FF'
        });
        // -- string
        css.addClass('lxCR-js-str', {
            color: '#E6DB5A !important'
        });
        // -- comment
        css.addClass('lxCR-js-cmt', {
            color: '#8D8872 !important'
        });

        // PHP
        // -- reserved
        css.addClass('lxCR-php-rsv', {
            fontWeight: 'bold',
            color: '#DE3427'
        });
        // -- symbols
        css.addClass('lxCR-php-smb', {
            color: '#DE3427'
        });
        // -- fucntion
        css.addClass('lxCR-php-fnc', {
            color: '#66D9D0'
        });
        // -- constructors
        css.addClass('lxCR-php-cns', {
            fontStyle: 'italic',
            color: '#66D9D0'
        });
        // -- variable
        css.addClass('lxCR-php-var', {
            fontStyle: 'italic',
            color: '#b0b0FF'
        });
        // -- constant
        css.addClass('lxCR-php-cst', {
            color: '#AE81FF'
        });
        // -- numbers
        css.addClass('lxCR-php-nmr', {
            color: '#DE91FF'
        });
        // -- string
        css.addClass('lxCR-php-str', {
            color: '#E6DB5A !important'
        });
        // -- comment
        css.addClass('lxCR-php-cmt', {
            color: '#8D8872 !important'
        });
    }

    constructor(box) {
        super();

        this.lang = null;
        this.container = box;
        this.textbox = new lx.Box({
            key: 'textbox',
            parent: box,
            style: {padding: '5px'},
            // left: '30px'  //todo - добавить нумерацию строк
        });
        this.context = new EditorContext(this);

        __init(this);
    }
    
    setLang(lang) {
        this.lang = lang;
    }

    getElement() {
        return this.textbox.getDomElem();
    }

    setText(text) {
        cr.context = this.context;
        var _c = this.getElement();

        _c.innerHTML = cr.markText(text) + '<span><br></span>';

        cr.loading(true);
        for (var i=0, l=_c.children.length; i<l; i++) {
            cr.marker.loadingCheck(cr.span( _c.children[i] ));
        }
        cr.loading(false);
        cr.context = null;
    }

    getText() {
        var str = this.textbox.html();

        str = str.replace(/<span.*?>/g, '');
        str = str.replace(/<\/span>/g, '');
        str = str.replace(/<br>$/, '');
        str = str.replace(/<br>/g, lx.EOL);

        str = str.replace(/&lt;/g, '<');
        str = str.replace(/&gt;/g, '>');
        str = str.replace(/&amp;/g, '&');

        return str;
    }

    markText(text) {
        // в месте перевода строки может стоять два символа - 13 и 10. 13 - это '\r', 10 не нужен
        text = text.replace( new RegExp(String.fromCharCode(13) + '?' + String.fromCharCode(10), 'g'), String.fromCharCode(13) );

        // разбил по переносам чтобы проще было обернуть пробельные символы
        var boof = text.split(String.fromCharCode(13)),
            ps = cr.lang().pareSymbols;

        // обернуть спанами все кроме служебных символов
        for (var i=0, l=boof.length; i<l; i++) {
            // оборачиваются слова
            boof[i] = boof[i].replace( /([\wа-яё\d]+)/gi, '<span>$1</span>' );
            // оборачиваются пробельные символы
            boof[i] = boof[i].replace( /(\s+)/g, '<span>$1</span>' );
        }
        // востанавливаю текст с заменой переносов строк
        boof = boof.join('<span><br></span>');

        // регулярное выражение для отлова парных служебных символов
        var psRe = '(';
        for (var i=0, l=ps.length; i<l; i++) {
            var temp = ps[i];
            temp = temp.replace(/(.)/g, '\\$1');
            if (i) psRe += '|';
            psRe += temp;
        }
        psRe += ')';
        psRe = new RegExp(psRe, 'g');

        // обернуть спанами служебные символы
        function overspan(p, pre, post) {
            var m = p.split(psRe);
            var res = pre;
            for (var i=0, l=m.length; i<l; i++) {
                if (i % 2) {
                    res += '<span>' + m[i] + '</span>';
                    continue;
                }
                if (m[i] == '') continue;
                for (var j=0, ll=m[i].length; j<ll; j++)
                    res += '<span>' + m[i][j] + '</span>';
            }
            res += post;
            return res;
        }
        // если он первый
        boof = boof.replace( /^([^\wа-яё\d\s]+?)<span>/i, function(str, p) { return overspan(p, '', '<span>') });
        // если он последний
        boof = boof.replace( /<\/span>([^\wа-яё\d\s]+?)$/i, function(str, p) { return overspan(p, '</span>', '') });
        // остальное
        boof = boof.replace( /<\/span>([^\wа-яё\d\s]+?)<span>/gi, function(str, p) { return overspan(p, '</span>', '<span>'); });

        return boof;
    }
}

function __init(self) {
    var _c = self.getElement();
    _c.setAttribute('contentEditable','true');
    _c.innerHTML = '<span><br></span>';
    _c.style.whiteSpace = 'pre';
    _c.style.overflow = 'visible';
    self.container.red = {
        key: null,
        lang: ''
    };
    self.container.context = self.context;
    self.textbox.on('focus', function(event) {
        _CodeEditor.context = self.context;
        self.container.red.key = null;
        self.context.hint.hide();
        self.container.on('keydown', __handlerKeydown);
        self.container.on('keyup', __handlerKeyup);
    });
    self.textbox.on('blur', function(event) {
        self.container.off('keydown', __handlerKeydown);
        self.container.off('keyup', __handlerKeyup);
        _CodeEditor.context = null;
    });
}

function __handlerKeydown(e) {
    const ctx = this.context;
    
    // то, что делается через CTRL
    if (lx.app.keyboard.ctrlPressed()) {
        // комментирование
        if (e.key == '/') ctx.getRange().lines().swapComment();

        // обработка вставки - чтобы заспанить вставляемый текст, нужно его перехватиь. Копирование идет стандартным путем, там спаны автоматом улетают
        if (e.keyCode == 86) ctx.getRange().paste();

        // ctrl-z
        if (e.keyCode == 90) ctx.history.back();

        return;
    }

    if (ctx.hint.isActive()) {
        if (e.keyCode == 37 || e.keyCode == 39 || e.keyCode == 27) ctx.hint.hide();
        else if (e.keyCode == 38 || e.keyCode == 40) {
            ctx.hint.move(e.keyCode);  // вверх 38, вниз 40
            e.preventDefault();
            return;
        }
    }

    // если удален символ
    if (e.keyCode == 8 || e.keyCode == 46) {
        e.preventDefault();
        ctx.getRange().del( e.keyCode );
        return;
    }

    if (e.keyCode == 13 || e.keyCode == 9) { e.preventDefault(); return; }

    if (e.key.length > 1) { this.red.key = null; return; }
    this.red.key = e.key.match( /[\wа-яё\s\d\[\]\(\){}\&\<\>\+\=\-\*\/\\\!\@\#\$\%\^\.\,\:\;\?\'\"\|]/i );

    if (this.red.key !== null) e.preventDefault();
}

function __handlerKeyup(e) {
    const ctx = this.context;

    // CTRL тут не нужен
    if (lx.app.keyboard.ctrlPressed()) return;
    // управление курсором стрелками тоже не надо
    if (37 <= e.keyCode && e.keyCode <= 40) return;

    // сразу обработаем ситуацию выбора подсказки
    if (ctx.hint.isActive() && e.keyCode == 13) {
        e.preventDefault();
        ctx.hint.choose();
        return;
    }

    // чтобы оставить энтер
    var key = e.key;
    if (e.keyCode == 13 || e.keyCode == 9) {
        key = (e.keyCode == 13) ? String.fromCharCode(13) : '\t';
        this.red.key = 1;
    }

    // остаются только символы и энтер
    if (key.length > 1) return;
    if (this.red.key == null) return;

    ctx.getRange().addChar( key );
}
