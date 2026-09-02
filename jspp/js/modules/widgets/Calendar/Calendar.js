// @lx:module lx.Calendar;
// @lx:module-data: i18n = i18n.yaml;

lx.import(
    lx.Input,
    lx.BasicCssContext
);

// @lx:<context CLIENT:
/*
 * Caches the calendar table - it is needed in a single copy, there are no situations
 * where several calendars are opened at the same time, respectively, the table is a singleton,
 * it is created on the first call, then it is taken from the cache
 * //TODO - this is the reason for css-scope BUG:
 *      Tables css-classes depends on lx.app.root and not connected with lx.Calendar instanse
 *      Need to refactor - ?make table rerendered every time?
 */
let _menu = null;
let _oldDate = null;
let _active = null;

let _dayTitles = [
    lx.i18n('monday'), lx.i18n('tuesday'), lx.i18n('wednesday'),
    lx.i18n('thursday'), lx.i18n('friday'), lx.i18n('saturday'), lx.i18n('sunday')
];

let _monthTitles = [
    lx.i18n('January'), lx.i18n('February'), lx.i18n('March'), lx.i18n('April'),
    lx.i18n('May'), lx.i18n('June'), lx.i18n('July'), lx.i18n('August'),
    lx.i18n('September'), lx.i18n('October'), lx.i18n('November'), lx.i18n('December')
];
// @lx:context>

/**
 * @widget lx.Calendar
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Calendar
 * - lx-Calendar-arroyL
 * - lx-Calendar-arroyR
 * - lx-Calendar-month
 * - lx-Calendar-dayTitle
 * - lx-Calendar-today
 * - lx-Calendar-daysTable
 * - lx-Calendar-day
 * - lx-Calendar-side-day
 * - lx-Calendar-current-day
 * - lx-Calendar-monthTable
 * - lx-Calendar-monthItem
 */
// @lx:namespace lx;
class Calendar extends lx.Input {
    static initCss(css) {
        css.useExtender(lx.BasicCssContext);
        css.inheritClass('lx-Calendar', 'Input');
        css.inheritClass('lx-Calendar-daysTable', 'AbstractBox', {
            color: css.presetValue('textColor', '#BABABA'),
            overflow: 'hidden'
        });
        css.inheritClass('lx-Calendar-monthTable', 'AbstractBox', {
            color: css.presetValue('textColor', '#BABABA'),
        });
        css.addClass('lx-Calendar-monthItem', {
            cursor: 'pointer'
        }, {
            hover: {
                backgroundColor: css.presetValue('checkedDarkColor', '#2f6a22'),
            }
        });
        css.addAbstractClass('lx-Calendar-arroy', {
            cursor: 'pointer',
            opacity: 0.5
        }, {
            hover: {
                opacity: 1
            }
        });
        css.inheritClasses({
            'lx-Calendar-arroyL' : { '@icon': ['\\2770', 16] },
            'lx-Calendar-arroyR': { '@icon': ['\\2771', 16] }
        }, 'lx-Calendar-arroy');
        css.inheritClass('lx-Calendar-month', 'Input', {
            cursor: 'pointer'
        });
        css.addClass('lx-Calendar-dayTitle', {
            background: css.presetValue('widgetGradient', 'linear-gradient(to bottom, #59574F, #403F3A)'),
            color: css.presetValue('widgetIconColor', '#DFD8B7')
        });
        css.addClass('lx-Calendar-today', {
            background: css.presetValue('widgetGradient', 'linear-gradient(to bottom, #59574F, #403F3A)'),
            color: css.presetValue('widgetIconColor', '#DFD8B7'),
            cursor: 'pointer'
        });
        css.addAbstractClass('lx-Calendar-every-day', {
            cursor: 'pointer'
        }, {
            hover: {
                backgroundColor: css.presetValue('checkedDarkColor', '#2f6a22')
            }
        });
        css.inheritClasses({
            'lx-Calendar-day': {},
            'lx-Calendar-side-day': { color: 'gray' }
        }, 'lx-Calendar-every-day');
        css.addClass('lx-Calendar-current-day', {
            background: css.presetValue('widgetGradient', 'linear-gradient(to bottom, #59574F, #403F3A)'),
            color: css.presetValue('widgetIconColor', '#DFD8B7')
        });
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Input::render::config),
     *     [date] {String}
     * }}
     */
    render(config) {
        this.addClass('lx-Calendar');
        this.date = config.date || (new Date()).toString();
        // @lx:<context CLIENT:
        this.date = new lx.Date(this.date);
        this.value( this.date.format() );
        // @lx:context>
    }

    // @lx:<context CLIENT:
    clientRender(config) {
        super.clientRender(config);
        this.on('mouseup', _handler_open);
        this.on('blur', _handler_blur);
    }

    postUnpack(config) {
        super.postUnpack(config);
        this.date = new lx.Date(this.date);
        this.value( this.date.format() );
    }
    // @lx:context>

    value(val) {
        if (val === undefined)
            return super.value();

        // @lx:<context CLIENT:
        this.date.reset(val);
        // @lx:context>
        // @lx:<context SERVER:
        this.date = val;
        // @lx:context>
        super.value(val);
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context CLIENT:
function _handler_open() {
    _active = this;
    _oldDate = this.date.format();
    _renew();

    let menu = _getMenu();
    menu.show();
    menu.satelliteTo(this);
    lx.on('mouseup', _lxHandler_outclick);
}

function _handler_blur() {
    this.date.reset(this.value());
}

function _lxHandler_outclick(event) {
    if (_active === null) return;
    event = event || window.event;

    let widget = event.target.__lx;
    if (widget === _active) return;
    if (widget && widget.ancestor({is:_menu})) return;
    _close(event);
}

function _close(event) {
    let menu = _getMenu();
    menu.hide();
    lx(menu)>>monthContainer.hide();
    lx.off('mouseup', _lxHandler_outclick);

    if (_active.date.format() != _oldDate)
        _active.trigger('change', event);
    _active = null;
}

function _renew() {
    let date = _active.date,
        list = _getMenu(),
        actDate = date.getDate();

    _active.value(date.format());
    lx(list)>>year.value(date.getFullYear());
    lx(list)>>month.text(_monthTitles[date.getMonth()]);

    let firstDay = new lx.Date(date.getYear(), date.getMonth(), 1);
    while (firstDay.getDay() !== 1)
        firstDay.shiftDay(-1);
    let lastDay = new lx.Date(date.getYear(), date.getMonth(), date.getMaxDate());
    while (lastDay.getDay() !== 0)
        lastDay.shiftDay(1);
    let weeksCount = Math.round((lastDay.getDaysBetween(firstDay) + 1) / 7);

    let days = lx(list)>>days;
    days.del('day', 28, days.childrenCount('day') - 28);
    if (weeksCount > 4) {
        let newDays = days.add(lx.Box, 7 * (weeksCount - 4), { key: 'day' });
        newDays.forEach(day=>{
            day.align(lx.CENTER, lx.MIDDLE);
            day.click(_handler_setDate);
        });
    }
    days.get('day').forEach(day=>{
        day.removeClass('lx-Calendar-day');
        day.removeClass('lx-Calendar-side-day');
        day.removeClass('lx-Calendar-current-day');
        day.addClass((firstDay.getMonth() == date.getMonth()) ? 'lx-Calendar-day' : 'lx-Calendar-side-day');
        if (firstDay.format() == date.format()) day.addClass('lx-Calendar-current-day');
        day.text(firstDay.getDate());
        day.__date = firstDay.format();
        firstDay.shiftDay(1);
    });

    lx(list)>>currentDay.text(lx.i18n('today') + ': ' + (new lx.Date).format());
}

function _handler_setDate(event) {
    let calendar = _active;
    calendar.date.reset(this.__date);
    calendar.value( calendar.date.format() );
    _close(event);
}

function _getMenu() {
    if (_menu) return _menu;

    const calendarMenu = new lx.Box({
        parent: lx.app.root,
        key: 'calendarMenu',
        geom: true,
        size: ['320px', '320px'],
        depthCluster: lx.DepthClusterMap.CLUSTER_FRONT
    });
    calendarMenu.overflow('visible');

    let tableContainer = calendarMenu.add(lx.Box, { geom: true, css: 'lx-Calendar-daysTable' });
    let monthContainer = calendarMenu.add(lx.Box,
        { geom: true, css: 'lx-Calendar-monthTable', key: 'monthContainer' }
    );

    tableContainer.stream();
    tableContainer.begin();
        let head = new lx.Box();
        head.grid({cols: 14, indent: '5px', minWidth: '10px'});
        head.add(lx.Box, { width: 1, css: 'lx-Calendar-arroyL', click:()=>{
            _active.date.shiftYear(-1);
            _renew();
        }});
        head.add(lx.Input, { width: 4, key: 'year' });
        head.add(lx.Box, { width: 1, css: 'lx-Calendar-arroyR', click:()=>{
            _active.date.shiftYear(1);
            _renew();
        }});
        head.add(lx.Box, { width: 1, css: 'lx-Calendar-arroyL', click:()=>{
            _active.date.shiftMonth(-1);
            _renew();
        }});
        head.add(lx.Box, { width: 6, key: 'month', css:'lx-Calendar-month' });
        head.add(lx.Box, { width: 1 , css: 'lx-Calendar-arroyR', click:()=>{
            _active.date.shiftMonth(1);
            _renew();
        }});

        let month = lx(head)>month;
        month.align(lx.CENTER, lx.MIDDLE);
        month.click(()=>monthContainer.show());

        let days = new lx.Box({key: 'days'});
        days.gridStream({cols:7});
        days.add(lx.Box, 7, { key: 'dayTitle', css: 'lx-Calendar-dayTitle' }, {
            preBuild: function (config, i) {
                config.text = _dayTitles[i];
                return config;
            }
        });
        days.get('dayTitle').forEach(box=>box.align(lx.CENTER, lx.MIDDLE));
        let newDays = days.add(lx.Box, 28, { key: 'day' });
        newDays.forEach(day=>{
            day.align(lx.CENTER, lx.MIDDLE);
            day.click(_handler_setDate);
        });
        (new lx.Box({ key: 'currentDay', css: 'lx-Calendar-today', click:(e)=>{
            _active.date.reset();
            _active.value(_active.date.format());
            _close(e);
        }})).align(lx.CENTER, lx.MIDDLE);
    tableContainer.end();

    monthContainer.gridProportional({cols: 2});
    monthContainer.add(lx.Box, 12, {css:'lx-Calendar-monthItem'}, {
        postBuild: function (el, i) {
            el.text(_monthTitles[i]);
            el.align(lx.CENTER, lx.MIDDLE);
            el.click(()=>{
                monthContainer.hide();
                _active.date.setMonth(i);
                _renew();
            });
        }
    });
    monthContainer.hide();

    _menu = calendarMenu;
    return calendarMenu;
}
// @lx:context>
