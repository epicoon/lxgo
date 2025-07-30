#lx:public;

#lx:use lx.Box;

class HintHolder {
    constructor(context) {
        this.context = context;
        this.activeHint = -1;
        this.isOn = true;
        this.minLen = 2;

        var h = 20; //context.editor.getElement().childNodes[0].offsetHeight;
        this.hintBox = new lx.Table({
            parent: context.editor.container,
            cols: 1,
            key: 'hint',
            size: ['240px', h*8+'px']
        });
        this.hintBox.hide()
            .style('color', 'black')
            .border({width: 0})
            .setRowsHeight(h+'!px')
            .setRowsCount(2);
    }
    
    isActive() {
        return this.hintBox.visibility();
    }

    isSwitchedOff() {
        return !this.isOn;
    }

    getMinLen() {
        return this.minLen;
    }

    select(num) {
        if ( this.activeHint != -1 ) this.hintBox.row(this.activeHint).removeClass('lxCR-hint');
        this.activeHint = num;
        if ( this.activeHint != -1 ) this.hintBox.row(this.activeHint).addClass('lxCR-hint');
    }

    show(l, t, info) {
        let hint = this.hintBox;
        hint.show();
        hint.coords( l+'px', t+'px' );

        hint.setRowsCount(0);
        hint.setRowsCount(info.length);
        hint.rows().forEach(row=>{
            row.on('mouseover', ()=>this.select(row.index));
            row.on('mousedown', ()=>this.choose());
        });

        hint.setContent(info, true);
        this.select(0);
    }

    hide() {
        this.select(-1);
        this.hintBox.hide();
    }

    move(dir) {
        if (dir == 38 && this.activeHint)
            this.select( this.activeHint - 1 );
        else if (dir == 40 && this.activeHint < this.hintBox.rowsCount() - 1)
        this.select( this.activeHint + 1 );
    }

    choose() {
        if (this.activeHint == -1) return;

        var text = this.hintBox.cell(this.activeHint, 0).text(),
            r = this.context.getRange(),
            s = r.anchor;

        this.context.trigger(EventType.HISTORY_INPUT, {
            span: s,
            info: {pos: s.len(), amt: text.length},
            force: true
        });

        r.anchor.setText(text);
        r.setCaret(s, s.len());

        this.hide();
    }
}
