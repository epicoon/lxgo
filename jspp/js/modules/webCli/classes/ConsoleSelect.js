#lx:public;

class ConsoleSelect {
    constructor(console, consoleBox, div, options, withQuit) {
        this.console = console;
        this.consoleBox = consoleBox;
        this.div = div;
        this.options = options;
        this.withQuit = withQuit;
        this.index = 0;
        this.pressed = false;

        this._onKeydown = this.onKeydown.bind(this);
        this._onKeyup = this.onKeyup.bind(this);
    }

    start() {
        this.render();
        this.consoleBox.on('keydown', this._onKeydown);
        this.consoleBox.on('keyup', this._onKeyup);
    }

    finish() {
        this.consoleBox.off('keydown', this._onKeydown);
        this.consoleBox.off('keyup', this._onKeyup);
    }
    
    render() {
        var text = '';
        this.options.forEach((elem, i)=>{
            text += '<div';
            if (i == this.index)
                text += ' class="lxWC-selected"'
            text += '>' + (i + 1) + '. ' + elem + '</div>';
        });
        if (this.withQuit) {
            text += (this.index == this.options.length)
                ? '<div class="lxWC-selected">q. Quit</div>'
                : '<div>q. Quit</div>';
        }
        this.div.innerHTML = text;
        this.console.actualizeScroll();
    }

    limit() {
        return (this.withQuit)
            ? this.options.length + 1
            : this.options.length;
    }

    onKeydown(e) {
        this.pressed = true;

        if (e.key == 'ArrowDown') {
            lx.preventDefault(e);
            if (this.index < this.limit() - 1) {
                this.index++;
                this.render();
            }
            return;
        }

        if (e.key == 'ArrowUp') {
            lx.preventDefault(e);
            if (this.index > 0) {
                this.index--;
                this.render();
            }
            return;
        }

        if (this.withQuit && e.key == 'q') {
            lx.preventDefault(e);
            this.index = this.options.length;
            this.render();
            return;
        }

        if (lx.isNumber(e.key) && +e.key <= this.options.length) {
            lx.preventDefault(e);
            this.index = +e.key - 1;
            this.render();
            return;
        }
    }

    onKeyup(e) {
        if (this.pressed && e.key == 'Enter') {
            lx.preventDefault(e);
            var result = this.index;
            if (this.withQuit && result == this.options.length) result = null;
            this.console.onSelected(e, result);
        }

        this.pressed = false;
    }
}
