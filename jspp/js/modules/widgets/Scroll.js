// @lx:module lx.Scroll;

/**
 * @widget lx.Scroll
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Scroll
 * - lx-Scroll-back
 * - lx-Scroll-handle-back
 * - lx-Scroll-handle
 */
// @lx:namespace lx;
class Scroll extends lx.Box {
    // @lx:const DEFAULT_SIZE = '15px';
    
    static initCss(css) {
        let scrollSize = parseInt(this.DEFAULT_SIZE, 10),
            trackPadding = Math.floor(scrollSize / 3),
            scrollBorderRadius = Math.round(scrollSize * 0.5) + 'px',
            scrollTrackPadding = trackPadding + 'px',
            scrollTrackBorderRadius = Math.round((scrollSize - trackPadding * 2) * 0.5) + 'px';
        css.addClass('lx-Scroll', {});
        css.addClass('lx-Scroll-back', {
            backgroundColor: css.preset.widgetIconColor,
            borderRadius: scrollBorderRadius,
            opacity: 0
        });
        css.addStyle('.lx-Scroll:hover .lx-Scroll-back', {
            opacity: 0.2,
            transition: 'opacity 0.3s linear'
        });
        css.addClass('lx-Scroll-handle-back', {
            padding: scrollTrackPadding
        });
        css.addClass('lx-Scroll-handle', {
            width: '100%',
            height: '100%',
            borderRadius: scrollTrackBorderRadius,
            backgroundColor: css.preset.widgetIconColor,
            opacity: 0.3
        });
        css.addStyle('.lx-Scroll-handle-back:hover .lx-Scroll-handle', {
            opacity: 0.6,
            transition: 'opacity 0.3s linear'
        });
    }

    modifyConfigBeforeApply(config) {
        if (config.parent === undefined && config.target)
            config.parent = config.target;
        if (config.target === undefined && config.parent)
            config.target = config.parent;

        config.type = config.type || lx.VERTICAL;
        if (config.type == lx.VERTICAL) {
            config.geom = [null, 0, lx.self(DEFAULT_SIZE), null, 0, lx.self(DEFAULT_SIZE)];
        } else {
            config.geom = [0, null, null, lx.self(DEFAULT_SIZE), lx.self(DEFAULT_SIZE), 0];
        }

        return config;
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     target {lx.Box},
     *     [type = lx.VERTICAL] {Number&Enum(
     *         lx.VERTICAL,
     *         lx.HORIZONTAL
     *     )}
     * }}
     */
    render(config) {
        if (!config.target || config.target === config.target.getContainer())
            throw 'Unavailable target for the Scroll widget';

        this.addClass('lx-Scroll');
        this.type = config.type || lx.VERTICAL;
        this.target = config.target;
        this.target.overflow('hidden');
        this.target.getContainer().overflow('hidden');
        this.add(lx.Box, {key: 'back', geom: true, css: 'lx-Scroll-back'});
        let handle = this.add(lx.Box, {
            key: 'handle',
            geom: [0, 0, this.type==lx.VERTICAL?'100%':'50%', this.type==lx.VERTICAL?'50%':'100%'],
            css: 'lx-Scroll-handle-back'
        });
        handle.add(lx.Box, {css: 'lx-Scroll-handle'});
    }

    isVertical() {
        return this.type == lx.VERTICAL;
    }

    // @lx:<context SERVER:
    beforePack() {
        this.target = this.target.renderIndex;
    }
    // @lx:context>

    // @lx:<context CLIENT:
    clientRender(config) {
        super.clientRender(config);

        _actualizeHandleSize(this);
        lx(this)>handle.move();

        lx(this)>handle.on('move', function() {
            if (this.parent.isVertical()) {
                let shift = this.top('px') / (this.parent.height('px') - this.height('px'));
                this.parent.target.scrollTo({yShift: shift});
            } else {
                let shift = this.left('px') / (this.parent.width('px') - this.width('px'));
                this.parent.target.scrollTo({xShift: shift});
            }
        });

        if (this.isVertical()) {
            this.target.on('wheel', (e)=>{
                let pos = this.target.getScrollPos();
                this.target.scrollTo({y: pos.y + e.deltaY});
            });
        }

        lx(this)>back.on('mousedown', (e)=>{
            if (e.target !== lx(this)>back.getDomElem()) return;

            let left, top;
            if (this.isVertical()) {
                let h = lx(this)>handle.height('px'),
                    h05 = Math.round(h * 0.5);
                top = e.offsetY - h05;
                if (top < 0) top = 0;
                else if (top + h > this.height('px')) top = this.height('px') - h;
            } else {
                let w = lx(this)>handle.width('px'),
                    w05 = Math.round(w * 0.5);
                left = e.offsetX - w05;
                if (left < 0) left = 0;
                else if (left + w > this.width('px')) left = this.width('px') - w;
            }

            let scrollSize = this.target.getScrollSize();
            if (this.isVertical())
                this.target.scrollTo({y: Math.round((top * scrollSize.height) / this.height('px'))});
            else
                this.target.scrollTo({x: Math.round((left * scrollSize.width) / this.width('px'))});
        });

        _handler_onResize(this);
        this.target.getContainer().on('contentResize', ()=>_handler_onResize(this));
        this.target.getContainer().on('resize', ()=>_handler_onResize(this));
        this.target.getContainer().on('scroll', ()=>_actualizeHandlePos(this));
    }

    restoreLinks(loader) {
        this.target = loader.getWidget(this.target);
    }

    moveTo(pos) {
        if (this.isVertical())
            this.target.scrollTo({y: pos});
        else
            this.target.scrollTo({x: pos});
    }
    // @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _actualizeHandle(self) {
    _actualizeHandleSize(self);
    _actualizeHandlePos(self);
}

function _actualizeHandleSize(self) {
    let c = self.target.getContainer(),
        scrollSize = self.target.getScrollSize();
    if (self.isVertical()) {
        let h = Math.floor((c.height('px') * self.height('px')) / scrollSize.height);
        if (h < 25) h = 25;
        lx(self)>handle.height(h + 'px');
    } else {
        let w = Math.floor((c.width('px') * self.width('px')) / scrollSize.width);
        if (w < 25) w = 25;
        lx(self)>handle.width(w + 'px');
    }
}

function _actualizeHandlePos(self) {
    let scrollSize = self.target.getScrollSize(),
        scrollPos = self.target.getScrollPos();
    if (self.isVertical()) {
        let t = Math.floor((self.height('px') * scrollPos.y) / scrollSize.height);
        lx(self)>handle.top(t + 'px');
    } else {
        let w = Math.floor((self.width('px') * scrollPos.x) / scrollSize.width);
        lx(self)>handle.left(w + 'px');
    }
}

function _handler_onResize(self) {
    let show = self.target.hasOverflow(self.type);
    self.visibility(show);
    if (show) _actualizeHandle(self);
}
