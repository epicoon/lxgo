// #lx:module lx.SmoothOpener;

// #lx:namespace lx;
class SmoothOpener extends lx.Element {
    static initCss(css) {
        css.addClass('lx-smooth-open', {
            animationName: 'lx-smooth-opener',
            animationDuration: '0.5s',
            animationIterationCount: '1'
        });
        css.addStyle('@keyframes lx-smooth-opener', {
            from: { opacity: 0 },
            to: { opacity: 1 }
        });
        css.addClass('lx-smooth-close', {
            animationName: 'lx-smooth-closer',
            animationDuration: '0.5s',
            animationIterationCount: '1'
        });
        css.addStyle('@keyframes lx-smooth-closer', {
            from: { opacity: 1 },
            to: { opacity: 0 }
        });
    }

    static apply(widget) {
        let f = widget.hide;
        widget.hide = function () {
            f.call(widget, 500);
        };

        widget.on('beforeShow', function () {
            this.removeClass('lx-smooth-close');
            this.addClass('lx-smooth-open');
        });
        widget.on('beforeHide', function () {
            this.removeClass('lx-smooth-open');
            this.addClass('lx-smooth-close');
        });
    }
}
