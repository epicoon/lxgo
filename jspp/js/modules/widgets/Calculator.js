// @lx:module lx.Calculator;

// @lx:use lx.Input;
// @lx:use lx.Button;

/**
 * @widget lx.Calculator
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-calc-abut
 * - lx-calc-cbut
 * - lx-calc-nbut
 * - lx-calc-rbut
 */
// @lx:namespace lx;
class Calculator extends lx.Box {
    static initCss(css) {
        css.addClass('lx-calc-nbut', {
            backgroundColor: css.preset.textBackgroundColor + ' !important'
        });
        css.addClass('lx-calc-cbut', {
            backgroundColor: css.preset.coldDeepColor + ' !important'
        });
        css.addClass('lx-calc-abut', {
        });
        css.addClass('lx-calc-rbut', {
            backgroundColor: css.preset.checkedDeepColor + ' !important'
        });
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [grid] {Object: #schema(lx.GridPositioningStrategy::applyConfig::config)}
     * }}
     */
    render(config) {
        super.render(config);

        let gridConfig = config.grid || {
            indent: '10px'
        };
        gridConfig.cols = 4;
        this.gridProportional(gridConfig);

        this.add(lx.Input, {key:'input', width: 4});
        let buts = [
            ['(', 'lx-calc-abut'], [')', 'lx-calc-abut'],
            ['<', 'lx-calc-cbut'], ['CE', 'lx-calc-cbut'],
            ['7', 'lx-calc-nbut'], ['8', 'lx-calc-nbut'],
            ['9', 'lx-calc-nbut'], ['+', 'lx-calc-abut'],
            ['4', 'lx-calc-nbut'], ['5', 'lx-calc-nbut'],
            ['6', 'lx-calc-nbut'], ['-', 'lx-calc-abut'],
            ['1', 'lx-calc-nbut'], ['2', 'lx-calc-nbut'],
            ['3', 'lx-calc-nbut'], ['*', 'lx-calc-abut'],
            ['0', 'lx-calc-nbut'], ['.', 'lx-calc-nbut'],
            ['=', 'lx-calc-rbut'], ['/', 'lx-calc-abut']
        ];
        for (let i in buts) {
            let text = buts[i][0],
                css = buts[i][1];
            this.add(lx.Button, {key:'but', text, css});
        }
    }

    // @lx:<context CLIENT:
    clientRender(config) {
        super.clientRender(config);
        let handlers = [
            _inpup, _inpup, _backspace, _clear,
            _inpup, _inpup, _inpup, _inpup,
            _inpup, _inpup, _inpup, _inpup,
            _inpup, _inpup, _inpup, _inpup,
            _inpup, _inpup, _result, _inpup,
        ];
        lx(this)>but.forEach((but, i)=>{
            but.click(()=>handlers[i](this, but));
        });
    }
    // @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _inpup(self, but) {
    let val = lx(self)>input.value(),
        char = but.text(),
        last = val.slice(-1);
    if (last != '') {
        let t1 = (last.match(/[\d\.\(\)]/)) ? 1 : 0,
            t2 = (char.match(/[\d\.\(\)]/)) ? 1 : 0;
        if (t1 != t2) val += ' ';
    }
    val += char;
    lx(self)>input.value(val);
}

function _backspace(self, but) {
    let val = lx(self)>input.value();
    if (val.length == 0) return;
    val = val.slice(0, -1);
    val = val.trim();
    lx(self)>input.value(val);
}

function _clear(self, but) {
    lx(self)>input.value('');
}

function _result(self, but) {
    let val = lx(self)>input.value();
    if (val == '') return;
    lx(self)>input.value(lx.Math.parseToCalculate('=' + val));
}
