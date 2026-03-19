// @lx:module lx.Paginator;
// @lx:module-data: i18n = i18n.yaml;

// @lx:use lx.BasicCssContext;

/**
 * @widget lx.Paginator
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Paginator
 * - lx-Paginator-to-start
 * - lx-Paginator-to-finish
 * - lx-Paginator-to-left
 * - lx-Paginator-to-right
 * - lx-Paginator-middle
 * - lx-Paginator-page
 * - lx-Paginator-active
 * - lx-Paginator-alt
 * 
 * Events:
 * - change
 * 
 * Pages start with 1 (not 0)
 */
// @lx:namespace lx;
class Paginator extends lx.Box {
	// @lx:const DEFAULT_SLOTS_COUNT = 7;
	// @lx:const DEFAULT_ELEMENTS_PER_PAGE = 10;

    static initCss(css) {
        css.useExtender(lx.BasicCssContext);
        css.addClass('lx-Paginator', {
            gridTemplateRows: '100% !important',
            overflow: 'hidden',
            color: css.preset.textColor,
            whiteSpace: 'nowrap',
            textOverflow: 'ellipsis',
            border: 'solid 1px ' + css.preset.widgetBorderColor,
            borderRadius: css.preset.borderRadius
        });
        css.addClass('lx-Paginator-middle', {
            width: 'auto'
        });
        css.addClass('lx-Paginator-page', {
            cursor: 'pointer'
        });
        css.addAbstractClass('Paginator-button', {
            background: css.preset.widgetGradient,
            color: css.preset.widgetIconColor,
            cursor: 'pointer'
        });
        css.inheritClass(
            'lx-Paginator-active',
            'Paginator-button',
            { borderRadius: css.preset.borderRadius }
        );
        css.inheritClasses({
            'lx-Paginator-to-finish': { '@icon': ['\\00BB', {paddingBottom:'10px'}] },
            'lx-Paginator-to-start' : { '@icon': ['\\00AB', {paddingBottom:'10px'}] },
            'lx-Paginator-to-left'  : { '@icon': ['\\2039', {paddingBottom:'10px'}] },
            'lx-Paginator-to-right' : { '@icon': ['\\203A', {paddingBottom:'10px'}] }
        }, 'Paginator-button');
        css.addClass('lx-Paginator-alt', {
            cursor: 'pointer',
        });
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [slotsCount = lx.Paginator.DEFAULT_SLOTS_COUNT] {Number},
     *     [elementsPerPage = lx.Paginator.DEFAULT_ELEMENTS_PER_PAGE] {Number},
     *     [elementsCount = 0] {Number},
     *     [activePage = 1] {Number}
     * }}
     */
    render(config) {
		super.render(config);

        this.addClass('lx-Paginator');
		this.firstSlotIndex = 0;
        this._elementsCount = lx.getFirstDefined(config.elementsCount, 0);
        this._elementsPerPage = config.elementsPerPage || lx.self(DEFAULT_ELEMENTS_PER_PAGE);
        this._pagesCount = Math.ceil(this._elementsCount / this._elementsPerPage);

		this.slotsCount = lx.getFirstDefined(config.slotsCount, lx.self(DEFAULT_SLOTS_COUNT));
        if (this.slotsCount <= 4) this.slotsCount = 1;
        this.slotsCountBase = this.slotsCount;
		_normalizeSlotsCount(this);

        this.runBuild();
        this.selectPage(lx.getFirstDefined(config.activePage, 1));
	}

    // @lx:<context CLIENT:
    clientRender(config) {
        super.clientRender(config);
        lx(this)>toStart.click(lx.self(toFirstPage));
        lx(this)>toLeft.click(lx.self(toPrevPage));
        lx(this)>toRight.click(lx.self(toNextPage));
        lx(this)>toFinish.click(lx.self(toLastPage));

        let middle = lx(this)>middle;
        if (middle.childrenCount() > 1) {
            middle.getChildren().forEach((a, i)=>{
                if (lx(a)>text.value() !== '...') a.click(lx.self(onSlotClick));
            });
        }
    }

    static onSlotClick(event) {
        this.ancestor({is:lx.Paginator}).selectPage(+(lx(this)>text.value()));
    }

    static toPrevPage(e) {
        let p = this.parent;
        e = e || p.newEvent();
        e.previousPage = p.activePage;
        p.selectPage(p.activePage - 1);
        e.currentPage = p.activePage;
        p.trigger('change', e);
    }

    static toNextPage(e) {
        let p = this.parent;
        e = e || p.newEvent();
        e.previousPage = p.activePage;
        p.selectPage(p.activePage + 1);
        e.currentPage = p.activePage;
        p.trigger('change', e);
    }

    static toFirstPage(e) {
        let p = this.parent;
        e = e || p.newEvent();
        e.previousPage = p.activePage;
        p.selectPage(1);
        e.currentPage = p.activePage;
        p.trigger('change', e);
    }

    static toLastPage(e) {
        let p = this.parent;
        e = e || p.newEvent();
        e.previousPage = p.activePage;
        p.selectPage(p._pagesCount);
        e.currentPage = p.activePage;
        p.trigger('change', e);
    }
    // @lx:context>

    elementsCount(cnt) {
        if (cnt === undefined) return this._elementsCount;

        this._elementsCount = cnt;
        this._pagesCount = Math.ceil(this._elementsCount / this._elementsPerPage);
        _normalizeSlotsCount(this);

        this.selectPage(this.activePage);
    }

    currentPage() {
        return this.activePage;
    }
    
    elementsPerPage() {
        return this._elementsPerPage;

        //TODO change this._elementsPerPage
    }

    pagesCount() {
        return this._pagesCount;
    }

	selectPage(number) {
        this.activePage = _validatePageNumber(this, number);

	    if (this.slotsCount == 1) _fillMiddleSimple(this);
        else if (this.slotsCount == 5 || this.slotsCount == 6) _fillMiddleMin(this);
        else _fillMiddleMax(this);
	}

    value(val = null) {
        if (val === null) return this.activePage;
        this.selectPage(val);
    }

    runBuild() {
	    this.streamProportional({direction: lx.HORIZONTAL, width: null});
	    this.begin();
	        //TODO 40px по задумке должны брыться из CSS, но не работает!
            new lx.Box({key: 'toStart', width:'40px', css: 'lx-Paginator-to-start'});
            new lx.Box({key: 'toLeft', width:'40px', css: 'lx-Paginator-to-left'});
            new lx.Box({key: 'middle'});
            new lx.Box({key: 'toRight', width:'40px', css: 'lx-Paginator-to-right'});
            new lx.Box({key: 'toFinish', width:'40px', css: 'lx-Paginator-to-finish'});
        this.end();

        let middle = lx(this)>middle;
        middle.stream({
            direction: lx.HORIZONTAL,
            indent: '5px',
            width: null,
            minWidth: 0
        });
	}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _normalizeSlotsCount(self) {
    self.slotsCount = self.slotsCountBase;
    if (self.slotsCount == 1 || self._pagesCount >= 7) return;

    if (self._pagesCount == 6) {
        self.slotsCount = 6;
        return;
    }

    if (self._pagesCount == 5) {
        self.slotsCount = 5;
        return;
    }

    if (self._pagesCount < 5)
        self.slotsCount = 1;
}

function _validatePageNumber(self, number) {
    if (number < 1) number = 1;
    if (number > self._pagesCount) number = self._pagesCount;

    lx(self)>toStart.disabled(false);
    lx(self)>toLeft.disabled(false);
    lx(self)>toRight.disabled(false);
    lx(self)>toFinish.disabled(false);

    if (number == 1) {
        lx(self)>toStart.disabled(true);
        lx(self)>toLeft.disabled(true);
    }

    if (number == self._pagesCount) {
        lx(self)>toRight.disabled(true);
        lx(self)>toFinish.disabled(true);
    }

    return number;
}

function _fillMiddleSimple(self) {
    let middle = lx(self)>middle;
    if (middle.childrenCount() > 1) middle.clear();
    if (middle.childrenCount() == 0) middle.align(lx.CENTER, lx.MIDDLE);
    middle.text(
        lx.i18n('Page') + ' '
        + self.activePage + ' '
        + lx.i18n('of') + ' '
        + self._pagesCount
    );
}

function _fillMiddleMin(self) {
    _rebuildMiddle(self);
    _applyMiddleSequence(
        self,
        _calcSequence(self.activePage, self._pagesCount, self.slotsCount)
    );
}

function _fillMiddleMax(self) {
    _rebuildMiddle(self);
    let preseq = _calcSequence(self.activePage - 1, self._pagesCount - 2, self.slotsCount - 2);
    seq = [1];
    preseq.forEach(a => seq.push((a === null) ? null : a + 1));
    seq.push(self._pagesCount);
    _applyMiddleSequence(self, seq);
}

function _rebuildMiddle(self) {
    let middle = lx(self)>middle;
    if (middle.childrenCount() != self.slotsCount) middle.clear();
    if (middle.childrenCount() == 0) {
        let c = middle.add(lx.Box, self.slotsCount, {width:'auto'});
        c.forEach(a=>{
            a.align(lx.CENTER, lx.MIDDLE);
            a.addClass('lx-Paginator-page');
        });
    }    
}

function _applyMiddleSequence(self, seq) {
    let middle = lx(self)>middle;
    middle.getChildren().forEach((a, i)=>{
        a.text(seq[i] === null ? '...' : seq[i]);
        if (seq[i] !== null)
            a.toggleClassOnCondition(seq[i] == self.activePage, 'lx-Paginator-active', 'lx-Paginator-alt');
        // @lx:<context CLIENT:
        if (seq[i] === null) a.off('click');
        else a.click(lx.Paginator.onSlotClick);
        // @lx:context>
    });
}

function _calcSequence(pageNumber, pagesCount, slotsCount) {
    let result = new Array(slotsCount);

    if (pageNumber <= Math.ceil(slotsCount * 0.5)) {
        for (let i=0; i<slotsCount-1; i++) result[i] = i + 1;
        result[slotsCount - 1] = null;
        return result;
    }

    if ((pagesCount - pageNumber) < (slotsCount - 2)) {
        result[0] = null;
        for (let i=1; i<slotsCount; i++) result[i] = pagesCount - slotsCount + i + 1;
        return result;
    }

    result[0] = null;
    result[slotsCount - 1] = null;
    let activeIndex = Math.ceil((slotsCount - 2) * 0.5),
        firstPage = pageNumber - activeIndex;
    for (let i=1; i<slotsCount-1; i++) result[i] = firstPage + i;
    return result;
}
