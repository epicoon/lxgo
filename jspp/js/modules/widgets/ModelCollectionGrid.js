// @lx:module lx.ModelCollectionGrid;

// @lx:use lx.Paginator;
// @lx:use lx.Input;
// @lx:use lx.Checkbox;
// @lx:use lx.Scroll;
// @lx:use lx.BasicCssContext;

/**
 * @widget lx.ModelCollectionGrid
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-MCG
 * - lx-MCG-lPart
 * - lx-MCG-head
 * - lx-MCG-rowBack
 * - lx-MCG-ellipsis
 * 
 * Events:
 * - rowClick
 */
// @lx:namespace lx;
class ModelCollectionGrid extends lx.Box {
    // @lx:const DEFAULT_COLUMN_WIDTH = '200px';

    static initCss(css) {
        css.useHolder(lx.BasicCssContext);
        css.inheritClass('lx-MCG', 'AbstractBox', {
            color: css.preset.textColor
        });
        css.addClass('lx-MCG-lPart', {
            borderRight: 'thick double ' + css.preset.widgetBorderColor
        });
        css.addClass('lx-MCG-head', {
            backgroundColor: css.preset.altMainBackgroundColor,
            borderBottom: 'thick double ' + css.preset.widgetBorderColor
        });
        css.addClass('lx-MCG-rowBack', {
            backgroundColor: css.preset.altMainBackgroundColor
        });
        css.addClass('lx-MCG-ellipsis', {
            '@ellipsis': true,
        });
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [header = false] {Boolean},
     *     [footer = false] {Boolean},
     *     [paginator = true] {Boolean}
     * }}
     */
    render(config={}) {
        super.render(config);

        this.addClass('lx-MCG');
        this.totalCount = null;
        this.collection = null;
        this.columnSequence = [];
        this.lockedColumn = null;
        this.columnModifiers = {};

        if (config.paginator === undefined)
            config.paginator = true;

        this.streamProportional({indent:'5px', direction: lx.VERTICAL});

        //TODO
        if (config.header) this.add(lx.Box, {key: 'header'});
        _buildWrapper(this);
        if (config.paginator) this.add(lx.Paginator, {key: 'paginator', height: '40px'});
        //TODO
        if (config.footer) this.add(lx.Box, {key: 'footer'});
    }

    // @lx:<context CLIENT:
    clientRender(config) {
        super.clientRender(config);

        const rBody = lx(this)>>rBody;
        const lBody = lx(this)>>lBody;
        rBody.on('scroll', ()=>{
            lx(this)>>rBack.scrollTo({ y: rBody.getScrollPos().y });
            lx(this)>>lBack.scrollTo({ y: rBody.getScrollPos().y });
            lx(this)>>lBody.scrollTo({ y: rBody.getScrollPos().y });
            lx(this)>>rHead.scrollTo({ x: rBody.getScrollPos().x });
        });
        lBody.on('wheel', e=>rBody.get('scrollV').moveTo(lBody.getScrollPos().y + e.deltaY));
    }
    // @lx:context>

    setTotalCount(count) {
        this.totalCount = count;
    }

    setCollection(collection) {
        this.collection = collection;
        if (this.totalCount === null)
            this.totalCount = this.collection.len;
    }

    dropCollection() {
        if (!this.collection) return;

        lx(this)>>lBackStream.dropMatrix();
        lx(this)>>lStream.dropMatrix();
        lx(this)>>rBackStream.dropMatrix();
        lx(this)>>rStream.dropMatrix();
        lx(this)>>lBackStream.clear();
        lx(this)>>lStream.clear();
        lx(this)>>rBackStream.clear();
        lx(this)>>rStream.clear();

        lx(this)>>lHeadStream.clear();
        lx(this)>>rHeadStream.clear();

        lx(this)>>lPart.width(0);

        this.totalCount = null;
        this.collection = null;
        this.columnSequence = [];
        this.lockedColumn = null;
        this.columnModifiers = {};

        lx(this)>paginator.setElementsCount(0);
    }

    setLockedColumn(columnName) {
        this.lockedColumn = columnName;
    }

    getColumnSequence() {
        if (this.columnSequence.len === 0) {
            if (this.collection === null)
                throw 'The grid must have a collection';

            this.columnSequence = this.collection.modelClass.getSchema().getFieldNames();
        }

        return this.columnSequence;
    }

    setColumnSequence(sequence) {
        this.columnSequence = sequence;
    }

    addColumn(config) {
        const columnName = config.name || null;
        if (!columnName)
            throw 'New colunm require name';

        let sequence = this.getColumnSequence();
        if (config.before) {
            let index = sequence.indexOf(config.before);
            if (index === -1)
                throw 'The grid\'s collection doesn\'t have column ' + config.before;
            sequence.splice(index, 0, columnName);
        } else
            sequence.push(columnName);
        this.setColumnSequence(sequence);

        this.modifyColumn(columnName, config);
    }

    modifyColumn(columnName, config) {
        let modifier = this.getColumnModifier(columnName);
        if (config.definition !== undefined)
            modifier.definition = config.definition;
        if (config.widget !== undefined)
            modifier.widget = config.widget;
        if (config.render !== undefined)
            modifier.render = config.render;
        modifier.title = lx.getFirstDefined(config.title, columnName);
        this.columnModifiers[columnName] = modifier;
    }

    getColumnModifier(columnName) {
        let modifier = this.columnModifiers[columnName] || {};

        if (modifier.definition === undefined)
            modifier.definition = _getFieldDefinition(this, columnName);

        if (modifier.widget === undefined)
            modifier.widget = _getDefaultColumnWidget(modifier.definition.type);
        if (modifier.widget.width === undefined)
            modifier.widget.width = lx(STATIC).DEFAULT_COLUMN_WIDTH;

        if (modifier.render === undefined)
            modifier.render = _getDefaultColumnRender(modifier.definition.type);

        if (modifier.title === undefined)
            modifier.title = columnName;

        return modifier;
    }

    rowAddClass(rowIndex, cssClass) {
        const lRow = lx(this)>>lBackStream.child(rowIndex);
        lRow.addClass(cssClass);
        const rRow = lx(this)>>rBackStream.child(rowIndex);
        rRow.addClass(cssClass);
    }

    rowRemoveClass(rowIndex, cssClass) {
        const lRow = lx(this)>>lBackStream.child(rowIndex);
        lRow.removeClass(cssClass);
        const rRow = lx(this)>>rBackStream.child(rowIndex);
        rRow.removeClass(cssClass);
    }

    getCell(columnKey, rowIndex) {
        const lRow = lx(this)>>lStream.child(rowIndex);
        if (lRow.contains(columnKey))
            return lRow.get(columnKey);

        const rRow = lx(this)>>rStream.child(rowIndex);
        return rRow.get(columnKey);
    }

    rerender() {
        if (!this.collection) return;

        const _t = this;
        const schema = this.collection.modelClass.getSchema();
        const sequence = this.getColumnSequence();
        const unlockedIndex = (this.lockedColumn)
            ? sequence.indexOf(this.lockedColumn) + 1
            : 0;
        const lHeadStream = lx(this)>>lHeadStream;
        const rHeadStream = lx(this)>>rHeadStream;

        // Left part
        let lockWidth = [];
        for (let i=0, l=unlockedIndex; i<l; i++) {
            let fieldName = sequence[i],
                columnModifier = this.getColumnModifier(fieldName);
            lockWidth.push(columnModifier.widget.width);
            const title = lHeadStream.add(lx.Box, {
                width: columnModifier.widget.width,
                text: columnModifier.title,
                css: 'lx-MCG-ellipsis'
            });
            title.align(lx.CENTER, lx.MIDDLE);
        }
        lockWidth = lx.Geom.calculate('+', ...lockWidth);
        lx(this)>>lPart.width(lockWidth);
        lx(this)>>lBackStream.matrix({
            items: this.collection,
            itemBox: [lx.Box, {css: 'lx-MCG-rowBack'}]
        });
        lx(this)>>lStream.matrix({
            items: this.collection,
            itemBox: [lx.Box, {stream: {direction: lx.HORIZONTAL}}],
            itemRender: (row, model) => {
                for (let i=0, l=unlockedIndex; i<l; i++) {
                    let fieldName = sequence[i],
                        columnModifier = _t.getColumnModifier(fieldName);
                    let config = {width: columnModifier.widget.width, css: 'lx-MCG-ellipsis'};
                    if (schema.hasField(fieldName))
                        config.field = fieldName;
                    else config.key = fieldName;
                    const box = new lx.Box(config);
                    box.align(lx.CENTER, lx.MIDDLE);
                    if (columnModifier.render)
                        columnModifier.render(box, model);
                }

                row.click(function (e) {
                    e.rowIndex = this.index;
                    _t.trigger('rowClick', e);
                });
            }
        });

        // Right part
        for (let i=unlockedIndex, l=sequence.len; i<l; i++) {
            let fieldName = sequence[i],
                columnModifier = this.getColumnModifier(fieldName);
            const title = rHeadStream.add(lx.Box, {
                width: columnModifier.widget.width,
                text: columnModifier.title,
                css: 'lx-MCG-ellipsis'
            });
            title.align(lx.LEFT, lx.MIDDLE);
        }
        lx(this)>>rBackStream.matrix({
            items: this.collection,
            itemBox: [lx.Box, {css: 'lx-MCG-rowBack'}]
        });
        lx(this)>>rStream.matrix({
            items: this.collection,
            itemRender: (rowWrapper, model) => {
                let row = rowWrapper.add(lx.Box, {
                    height: '100%',
                    stream: {direction: lx.HORIZONTAL}
                });

                row.begin();
                for (let i=unlockedIndex, l=sequence.len; i<l; i++) {
                    let fieldName = sequence[i],
                        columnModifier = _t.getColumnModifier(fieldName);
                    let config = {width: columnModifier.widget.width, css: 'lx-MCG-ellipsis'};
                    if (schema.hasField(fieldName))
                        config.field = fieldName;
                    else config.key = fieldName;
                    const box = new lx.Box(config);
                    box.align(lx.LEFT, lx.MIDDLE);
                    if (columnModifier.render)
                        columnModifier.render(box, model);
                }
                row.end();

                rowWrapper.click(function (e) {
                    e.rowIndex = this.index;
                    _t.trigger('rowClick', e);
                });
            }
        });

        lx(this)>paginator.setElementsCount(this.totalCount);
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _buildWrapper(self) {
    const gridWrapper = self.add(lx.Box, {key: 'wrapper'});
    gridWrapper.streamProportional({direction: lx.HORIZONTAL});
    gridWrapper.begin();

    let streamConfig = {indent: '5px'};

    const lPart = new lx.Box({
        key: 'lPart',
        width: '0px',
        css: 'lx-MCG-lPart'
    });
    lPart.style('min-width', '0px');

    const lHead = lPart.add(lx.Box, {
        key: 'lHead',
        geom: [0, 0, 100, '50px'],
        css: 'lx-MCG-head'
    });
    const lBack = lPart.add(lx.Box, {
        key: 'lBack',
        geom: [0, '50px', null, null, 0, 0]
    });
    const lBody = lPart.add(lx.Box, {
        key: 'lBody',
        geom: [0, '50px', null, null, 0, 0]
    });
    lHead.add(lx.Box, {key: 'lHeadStream', height: '100%', stream: {direction: lx.HORIZONTAL}});
    lBack.overflow('hidden');
    lBody.overflow('hidden');

    const lBackStream = lBack.add(lx.Box, {key: 'lBackStream'});
    lBackStream.stream(streamConfig);
    const lStream = lBody.add(lx.Box, {key: 'lStream'});
    lStream.stream(streamConfig);


    const rPart = new lx.Box({key: 'rPart'});

    const rHead = rPart.add(lx.Box, {
        key: 'rHead',
        geom: [0, 0, 100, '50px'],
        css: 'lx-MCG-head'
    });
    const rBack = rPart.add(lx.Box, {
        key: 'rBack',
        geom: [0, '50px', null, null, 0, 0]
    });
    const rBody = rPart.add(lx.Box, {
        key: 'rBody',
        geom: [0, '50px', null, null, 0, 0]
    });
    rHead.add(lx.Box, {key: 'rHeadStream', height: '100%', stream: {direction: lx.HORIZONTAL}});
    rHead.overflow('hidden');
    rBack.overflow('hidden');
    rBody.addContainer();
    rBody.addStructure(lx.Scroll, {key:'scrollV', type: lx.VERTICAL});
    rBody.addStructure(lx.Scroll, {key:'scrollH', type: lx.HORIZONTAL});

    const rBackStream = rBack.add(lx.Box, {key: 'rBackStream'});
    rBackStream.stream(streamConfig);
    const rStream = rBody.add(lx.Box, {key: 'rStream'});
    rStream.stream(streamConfig);

    console.log(rStream);

    gridWrapper.end();
}

function _getFieldDefinition(self, fieldName) {
    const schema = self.collection.modelClass.getSchema();
    let fieldData = schema.hasField(fieldName) ? schema.getField(fieldName) : {};
    if (!fieldData.type)
        fieldData.type = lx.ModelTypeEnum.STRING;
    return fieldData;
}

function _getDefaultColumnWidget(type) {
    switch (type) {

        //TODO - старый способ представления первичного ключа на фронте. Требует рефакторинга
        case 'pk':

        case lx.ModelTypeEnum.INTEGER:
        case lx.ModelTypeEnum.BOOLEAN:
            return {width: '100px'};
        case lx.ModelTypeEnum.STRING:
            return {width: '200px'};
        default:
            return {width: '200px'};
    }
}

function _getDefaultColumnRender(type) {
    switch (type) {
        case lx.ModelTypeEnum.INTEGER:
        case lx.ModelTypeEnum.BOOLEAN:
        case lx.ModelTypeEnum.STRING:
            return null;
        default:
            return null;
    }
}
