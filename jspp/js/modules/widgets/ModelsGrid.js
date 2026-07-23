// @lx:module lx.ModelsGrid;

lx.import(
    lx.Paginator,
    lx.Input,
    lx.Checkbox
);

/**
 * @widget lx.ModelsGrid
 * @content-disallowed
 * 
 * Events:
 * - applyFilters
 * 
 * CSS classes:
 * - lx-ModelsGrid
 * - lx-ModelsGrid-header
 * - lx-ModelsGrid-footer
 * - lx-ModelsGrid-column
 * - lx-ModelsGrid-edge
 * - lx-ModelsGrid-af
 * - lx-ModelsGrid-rf
 * - lx-ModelsGrid-o
 * - lx-ModelsGrid-oc
 * - lx-ModelsGrid-oa
 * - lx-ModelsGrid-od
 * - lx-ModelsGrid-oaa
 * - lx-ModelsGrid-oda
 * - lx-ModelsGrid-fs
 * - lx-ModelsGrid-fsi
 * - lx-ModelsGrid-fc
 */
// @lx:namespace lx;
class ModelsGrid extends lx.Box {
    // @lx:const FILTER_NONE = '_';
    // @lx:const FILTER_EQUAL = '=';
    // @lx:const FILTER_NOT_EQUAL = '!=';
    // @lx:const FILTER_ASYMP = 'LIKE';
    // @lx:const FILTER_LESS = '<';
    // @lx:const FILTER_ELESS = '<=';
    // @lx:const FILTER_GREATER = '>';
    // @lx:const FILTER_EGREATER = '>=';

    static initCss(css) {
        css.addClass('lx-ModelsGrid', {
            backgroundColor: 'white',
        });
        css.addClass('lx-ModelsGrid-header', {
            backgroundColor: '#eeeeee',
        });
        css.addClass('lx-ModelsGrid-footer', {
            backgroundColor: '#eeeeee',
        });
        
        css.addClass('lx-ModelsGrid-column', {
            borderLeft: '1px solid lightgray',
            borderRight: '1px solid lightgray',
            backgroundColor: 'white',
        });
        css.addClass('lx-ModelsGrid-edge', {
            cursor: 'ew-resize',
        }, {
            hover: {
                backgroundColor: 'lightgray',
            }
        });

        // ⟲︎ &#10226; \27F2
        // ∅︎ &#8709;  \2205
        // Apply filters
        css.addClass('lx-ModelsGrid-af', {
            cursor: 'pointer',
            fontSize: '1.8em',
        }, {
            hover: {backgroundColor: 'lightgray'},
            before: {content: '"\\27F2"'}
        });
        // Reset filters
        css.addClass('lx-ModelsGrid-rf', {
            cursor: 'pointer',
            fontSize: '1.8em',
        }, {
            hover: {backgroundColor: 'lightgray'},
            before: {content: '"\\2205"'}
        });

        // Order
        css.addClass('lx-ModelsGrid-o', {
            cursor: 'pointer',
            userSelect: 'none',
        });
        // Order changed
        css.addClass('lx-ModelsGrid-oc', {
            backgroundColor: 'yellow',
        });
        // △ &#9651; \25B3
        // ▽ &#9661; \25BD
        // ▲ &#9650; \25B2
        // ▼ &#9660; \25BC
        // Order ASC
        css.addClass('lx-ModelsGrid-oa', {}, {before: {content: '"\\25B3"'}});
        // Order DESC
        css.addClass('lx-ModelsGrid-od', {}, {before: {content: '"\\25BD"'}});
        // Order ASC active
        css.addClass('lx-ModelsGrid-oaa', {fontSize:'larger'}, {before: {content: '"\\25B2"'}});
        // Order DESC active
        css.addClass('lx-ModelsGrid-oda', {fontSize:'larger'}, {before: {content: '"\\25BC"'}});

        // filter selector
        css.addClass('lx-ModelsGrid-fs', {
            backgroundColor: 'white',
            border: 'solid 1px lightgray',
        });

        // filter selector item
        css.addClass('lx-ModelsGrid-fsi', {
            cursor: 'pointer',
        }, {
            hover: {
                backgroundColor: 'lightgray',
            }
        });

        // filter changed
        css.addClass('lx-ModelsGrid-fc', {
            backgroundColor: 'yellow',
        });
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     stateKeeper {String}
     * }}
     */
    render(config={}) {
        super.render(config);

        this.addClass('lx-ModelsGrid');

        this.fcMap = {};
        if (config.stateKeeper)
            this.stateKeeper = config.stateKeeper;

        this.begin();
        lx.ml(`
        <lx.Box> [_]\
            #streamProportional()
            <lx.Box> @header.lx-ModelsGrid-header (height:'42px')
                <lx.Box> [_]\
                    #streamProportional(direction: lx.HORIZONTAL)
                    <lx.Box> @applyFilters .lx-ModelsGrid-af (width:'42px') #align(lx.CENTER, lx.MIDDLE)
                    <lx.Box> @resetFilters .lx-ModelsGrid-rf (width:'42px') #align(lx.CENTER, lx.MIDDLE)
                    <lx.Box>
                    <lx.Box> @total (width:'auto') "Total:" #align(lx.CENTER, lx.MIDDLE)
            <lx.Box> #overflow('auto')
                <lx.Box> @content (height:'auto')\
                    #stream(direction: lx.HORIZONTAL, minWidth:'4px')
            <lx.Box> @pgnWrapper (height:'50px')
                <lx.Paginator> @paginator [_]
            <lx.Box> @footer.lx-ModelsGrid-footer (height:'50px')
        `);
        this.end();
    }

    paginator() {
        return lx(this)>>paginator;
    }

    /**
     * @param {Number} count
     */
    setTotalCount(count) {
        this.paginator().elementsCount(count);
        lx(this)>>total.text('Total: ' + count);
    }

    // @lx:<context CLIENT:
    clientRender(config) {
        super.clientRender(config);
        this.stateKeeper = new State(this, this.stateKeeper);
        this.schema = null;
        this.customCols = {};

        lx(this)>>applyFilters.click(()=>{
            this.trigger('applyFilters', this.newEvent({
                filters: this.getFilters(),
            }));
            this.commitFilters();
        });
        lx(this)>>resetFilters.click(()=>{
            this.resetFilters();
        });
    }

    /**
     * @param config {Object: {
     *     key {String},
     *     header {String},
     *     cellRender {Function}
     * }}
     */
    setCustomColumn(config) {
        this.customCols[config.key] = config;
    }

    getFilters() {
        const contentBox = lx(this)>>content,
            columns = contentBox.findAll('column');

        let res = {};

        columns.forEach(col=>{
            if (!col.data || col.data.orderAsc === undefined) return;
            let setting = {};
            if (col.data.orderAsc)
                setting.order = 'ACS';
            else if (col.data.orderDesc)
                setting.order = 'DESC';
            if (!setting.lxEmpty())
                res[col.data.fieldName] = setting;

            const f = col.findOne('filter');
            if (f === null) return res;

            const operator = f.findOne('operator');
            if (!operator || !operator.data || operator.data.filter === undefined
                || operator.data.filter === lx.ModelsGrid.FILTER_NONE) return;

            const pattern = f.findOne('pattern');  
            if (!pattern) return res;

            if (!(col.data.fieldName in res))
                res[col.data.fieldName] = {};
            res[col.data.fieldName].operator = operator.data.filter;
            res[col.data.fieldName].pattern = pattern.value();
        });

        return res;
    }

    commitFilters() {
        const contentBox = lx(this)>>content,
            columns = contentBox.findAll('column');
        columns.forEach(col=>{
            const f = col.findOne('filter');
            if (f === null) return;

            const operator = f.findOne('operator');
            if (!operator || !operator.data || operator.data.filter === undefined)
                return;

            operator.data.origFilter = operator.data.filter;
            operator.parent.removeClass('lx-ModelsGrid-fc');
            if (operator.data.filter == lx.ModelsGrid.FILTER_NONE)
                lx(col)>>pattern.value('');

            col.data.orderAscOrig = col.data.orderAsc;
            col.data.orderDescOrig = col.data.orderDesc;
            lx(col)>>orderAsc.removeClass('lx-ModelsGrid-oc');
            lx(col)>>orderDesc.removeClass('lx-ModelsGrid-oc');
        });
    }

    resetFilters() {
        const contentBox = lx(this)>>content,
            columns = contentBox.findAll('column');
        columns.forEach(col=>{
            const f = col.findOne('filter');
            if (f === null) return;

            const operator = f.findOne('operator');
            if (!operator || !operator.data || operator.data.filter === undefined)
                return;

            operator.html('&#8709;');
            operator.data.filter = lx.ModelsGrid.FILTER_NONE;
            (operator.data.filter !== operator.data.origFilter)
                ? operator.parent.addClass('lx-ModelsGrid-fc')
                : operator.parent.removeClass('lx-ModelsGrid-fc');

            _setOrderAsc(col, false);
            _setOrderDesc(col, false);
        });
    }

    hasSchema() {
        return this.schema !== null;
    }

    setSchema(schema) {
        this.schema = schema;

        const state = this.stateKeeper.load();

        this.fcMap = {};
        let inx = 0, min = Infinity;
        schema.eachField((_, fieldName)=>{
            if (fieldName in state) {
                if (state[fieldName].v) {
                    this.fcMap[fieldName] = state[fieldName].i;
                    inx = Math.max(inx, state[fieldName].i);
                } else {
                    this.fcMap[fieldName] = null;
                }
            } else {
                this.fcMap[fieldName] = -1;
            }
        });
        for (let colKey in this.customCols) {
            if (colKey in state) {
                this.fcMap[colKey] = state[colKey].i;
                inx = Math.max(inx, state[colKey].i);
            } else {
                this.fcMap[colKey] = -1;
            }
        }
        inx++;
        let normalizer = {}, max = 0;
        for (let fieldName in this.fcMap) {
            if (this.fcMap[fieldName] === -1)
                this.fcMap[fieldName] = inx++;
            max = Math.max(max, this.fcMap[fieldName]);
            normalizer[this.fcMap[fieldName]] = fieldName;
        }

        let counter = 0;
        for (let i=0; i<=max; i++) {
            if (normalizer[i] === undefined) continue;
            this.fcMap[normalizer[i]] = counter++;
        }

        const contentBox = lx(this)>>content;
        for (let fieldName in this.fcMap) {
            if (this.fcMap[fieldName] === null) continue;

            let cw = contentBox.add(lx.Box, {
                key: 'columnWrapper',
                width: '150px',
            });
            let c = cw.add(lx.Box, {
                key: 'column',
                geom: true,
                css: 'lx-ModelsGrid-column',
            })
            c.style('position', 'relative');

            _initEdge(this, contentBox.add(lx.Box, {
                key: 'edge',
                width: '4px',
                css: 'lx-ModelsGrid-edge'
            }));
        }
    }

    /**
     * @param c {lx.ModelCollection}
     * @param [metaData] {Object: {
     *     [labels] {Object}
     * }}
     */
    setModels(c, metaData = {}) {
        if (!this.hasSchema())
            this.setSchema(c.getModelSchema());

        const state = this.stateKeeper.load(),
            contentBox = lx(this)>>content,
            columns = contentBox.findAll('column');
        this.schema.eachField((field, fieldName)=>{
            if (this.fcMap[fieldName] === null) return;

            let columnBox = columns.at(this.fcMap[fieldName]);
            columnBox.addData({
                fieldName,
                orderAsc: false,
                orderDesc: false,
                orderAscOrig: false,
                orderDescOrig: false,
            });
            if (fieldName in state)
                columnBox.parent.width((state[fieldName].w + 'px'));

            columnBox.begin();
            const cView = lx.ml(`
            <lx.Box> #stream()
                <lx.Box>
                    <lx.Box> @head [_] "${fieldName}" #align(lx.CENTER, lx.MIDDLE)
                    <lx.Box> [0:0:20px:100]
                        <lx.Box> @orderAsc.lx-ModelsGrid-o.lx-ModelsGrid-oa [0:0:100:50] #align(lx.CENTER, lx.BOTTOM)
                        <lx.Box> @orderDesc.lx-ModelsGrid-o.lx-ModelsGrid-od [0:50:100:50] #align(lx.CENTER, lx.TOP)
                <lx.Box> (height:'42px')
                    call: _renderFilter(field.type)
                <lx.Box> @fields #stream(direction: lx.VERTICAL, height:'42px')
            `);
            columnBox.end();

            cView.orderAsc.click(()=>{
                _setOrderDesc(columnBox, false);
                _setOrderAsc(columnBox, !columnBox.data.orderAsc);
            });
            cView.orderDesc.click(()=>{
                _setOrderAsc(columnBox, false);
                _setOrderDesc(columnBox, !columnBox.data.orderDesc);
            });

            _initColumnMove(this, lx(columnBox)>>head, columnBox, fieldName);

            lx(columnBox)>>fields.matrix({
                items: c,
                itemBox: [lx.Box, {geom: true}],
                itemRender: (row, model) => {
                    // row.border();

                    row.style('padding', '4px');
                    if (field.type == lx.ModelTypeEnum.BOOLEAN) {
                        let cb = row.add(lx.Checkbox, {
                            geom: true,
                            field: fieldName,
                        });
                        cb.setAttribute('readonly');
                    } else {
                        let inp = row.add(lx.Input, {
                            field: fieldName,
                        });
                        inp.setAttribute('readonly');
                    }
                }
            });
        });

        for (let colKey in this.customCols) {
            let conf = this.customCols[colKey];

            let columnBox = columns.at(this.fcMap[colKey]);
            if (colKey in state)
                columnBox.parent.width((state[colKey].w + 'px'));

            //TODO
            columnBox.border();

            columnBox.begin();
            const cView = lx.ml(`
            <lx.Box> #stream()
                <lx.Box>
                    <lx.Box> @head [_] "${conf.header}" #align(lx.CENTER, lx.MIDDLE)
                <lx.Box> (height:'42px')
                <lx.Box> @fields #stream(direction: lx.VERTICAL, height:'42px')
            `);
            columnBox.end();

            _initColumnMove(this, lx(columnBox)>>head, columnBox, colKey);

            lx(columnBox)>>fields.matrix({
                items: c,
                itemBox: [lx.Box, {geom: true}],
                itemRender: (row, model) => {
                    conf.cellRender(row, model);
                }
            });
        }
    }

    dropModels() {
        //TODO
    }
    // @lx:context>
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// @lx:<context CLIENT:

lx.ModelsGrid.opened = [];

/**
 * @private
 * @param {String} type
 */
function _renderFilter(type) {
    if (type == lx.ModelTypeEnum.BOOLEAN) {
        new lx.Box({text:'TODO'});
        return;
    }

    if (type == lx.ModelTypeEnum.NUMBER || type == lx.ModelTypeEnum.STRING) {
        // ∅︎ &#8709;
        // = &#61;
        // ≠︎ &#8800;
        // ≈︎ &#8776;
        // < &#60;
        // ≤︎ &#8804;
        // > &#62;
        // ≥︎ &#8805;
        const ftr = lx.ml(`
        <lx.Box> @filter [_] #streamProportional(direction:lx.HORIZONTAL, minWidth:'10px')
            <lx.Box> @operatorWrapper (width:'20px')
                <lx.Box> @operator [_] (html:'&#8709;') #align(lx.CENTER, lx.MIDDLE) #style('cursor', 'pointer')
                if type == lx.ModelTypeEnum.NUMBER:
                    <lx.Box> @selector.lx-ModelsGrid-fs (top:'100%', width:'40px') #style('z-index', 1000)\
                        #grid(minWidth:'10px', minHeight:'20px', cols:2) #hide()
                        <lx.Box> (html: '&#8709;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_NONE}
                        <lx.Box>
                        <lx.Box> (html: '&#61;'  ) #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_EQUAL}
                        <lx.Box> (html: '&#8800;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_NOT_EQUAL}
                        <lx.Box> (html: '&#60;'  ) #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_LESS}
                        <lx.Box> (html: '&#8804;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_ELESS}
                        <lx.Box> (html: '&#62;'  ) #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_GREATER}
                        <lx.Box> (html: '&#8805;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_EGREATER}
                else:
                    <lx.Box> @selector.lx-ModelsGrid-fs (top:'100%', width:'60px') #style('z-index', 1000)\
                        #grid(minWidth:'10px', minHeight:'20px', cols:3) #hide()
                        <lx.Box> (html: '&#8709;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_NONE}
                        <lx.Box>
                        <lx.Box>
                        <lx.Box> (html: '&#61;'  ) #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_EQUAL}
                        <lx.Box> (html: '&#8800;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_NOT_EQUAL}
                        <lx.Box> (html: '&#8776;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_ASYMP}
                        <lx.Box> (html: '&#60;'  ) #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_LESS}
                        <lx.Box> (html: '&#8804;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_ELESS}
                        <lx.Box>
                        <lx.Box> (html: '&#62;'  ) #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_GREATER}
                        <lx.Box> (html: '&#8805;') #align(lx.CENTER, lx.MIDDLE) {filter:lx.ModelsGrid.FILTER_EGREATER}
            <lx.Box>
                <lx.Input> @pattern (top:'4px', right:'4px', bottom:'4px') #style('width', 'auto')
        `);

        ftr.operator.data = {
            filter: lx.ModelsGrid.FILTER_NONE,
            origFilter: lx.ModelsGrid.FILTER_NONE,
        };

        ftr.selector.getChildren().forEach(el => {
            if (!el.data) return;
            el.addClass('lx-ModelsGrid-fsi');
        });

        ftr.operatorWrapper.click(e=>{
            if (e.target.__lx.parent !== e.targetWidget) return;
            ftr.selector.show();
            lx.ModelsGrid.opened.push(ftr.selector);
            setTimeout(()=>lx.on('click', _onClickFilter));
        });
    }

    //TODO new types
}

function _onClickFilter(e) {
    if (!e.target.__lx || !e.target.__lx.data || e.target.__lx.data.filter === undefined) {
        lx.off('click', _onClickFilter);
        lx.ModelsGrid.opened.forEach(e=>e.hide());
        lx.ModelsGrid.opened = [];
        return;
    }

    const selection = e.target.__lx,
        val = selection.parent.parent.childrenByKeys.operator;
    if (val.data.filter !== selection.data.filter) {
        val.data.filter = selection.data.filter;
        val.html(selection.html());
        val.data.filter === val.data.origFilter
            ? val.parent.removeClass('lx-ModelsGrid-fc')
            : val.parent.addClass('lx-ModelsGrid-fc');
    }

    lx.ModelsGrid.opened.forEach(e=>e.hide());
    lx.ModelsGrid.opened = [];
    lx.off('click', _onClickFilter);
}

/**
 * @private
 * @param {ModelsGrid} self
 * @param {lx.Box} edge
 */
function _initEdge(self, edge) {
    edge.onUp = _onUp.bind(edge);
    edge.onMove = _onMove.bind(edge);

    edge.on('mousedown', ()=>{
        edge.moveData = {
            width: lx(self)>>columnWrapper.at(edge.index).width('px'),
            x0: lx.app.mouse.x
        };
        lx.app.mouse.onUp(edge.onUp);
        lx.app.mouse.onMove(edge.onMove);
        document.body.style.userSelect = "none";
    });
}

function _onMove() {
    const c = lx(this.parent)>>columnWrapper.at(this.index);
    c.width((this.moveData.width + lx.app.mouse.x - this.moveData.x0) + 'px');
}

function _onUp() {
    document.body.style.userSelect = "";
    lx.app.mouse.offUp(this.onUp);
    lx.app.mouse.offMove(this.onMove);
    this.moveData = null;
    this.ancestor({is: lx.ModelsGrid}).stateKeeper.commit();
}

/**
 * @private
 * @param {ModelsGrid} self
 * @param {lx.Box} head
 * @param {lx.Box} column
 * @param {String} fieldName
 */
function _initColumnMove(self, head, column, fieldName) {
    head.move({
        parentMove: column,
        yMove: false,
        xLimit: false,
    });

    head.on('beforeMove', ()=>{
        column.style('z-index', 1000);
    });

    head.on('move', ()=>{
        const r = column.getGlobalRect(),
            middle = r.left + 0.5 * r.width;

        const slots = lx(self)>>columnWrapper;
        for (let iFieldName in self.fcMap) {
            if (iFieldName == fieldName) continue;
            const contrSlot = slots.at(self.fcMap[iFieldName]);

            let sr = contrSlot.getGlobalRect();

            if (middle < sr.left || middle > sr.left + Math.min(r.width, sr.width))
                continue;

            const currentSlot = column.parent,
                contrColumn = lx(contrSlot)>>column,
                w = currentSlot.width('px');

            contrColumn.setParent(null);
            column.setParent(contrSlot);
            contrColumn.setParent(currentSlot);
            column.left((r.left - sr.left) + 'px');

            currentSlot.width(contrSlot.width('px') + 'px');
            contrSlot.width(w + 'px');
            lx.app.dragNDrop.resetDelta(head);

            self.fcMap[fieldName] = contrSlot.index;
            self.fcMap[iFieldName] = currentSlot.index;
            break;
        }
    });

    head.on('moveEnd', ()=>{
        column.left(null);
        column.style('z-index', null);
        self.stateKeeper.commit();
    });
}

function _setOrderAsc(columnBox, val) {
    columnBox.data.orderAsc = val;
    const orderAsc = lx(columnBox)>>orderAsc;
    orderAsc.toggleClassOnCondition(val !== columnBox.data.orderAscOrig, 'lx-ModelsGrid-oc');
    orderAsc.toggleClassOnCondition(val, 'lx-ModelsGrid-oaa');
}

function _setOrderDesc(columnBox, val) {
    columnBox.data.orderDesc = val;
    const orderDesc = lx(columnBox)>>orderDesc;
    orderDesc.toggleClassOnCondition(val !== columnBox.data.orderDescOrig, 'lx-ModelsGrid-oc');
    orderDesc.toggleClassOnCondition(val, 'lx-ModelsGrid-oda');
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * State
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

class State {
    /**
     * @param {ModelsGrid} grid
     * @param {String} key
     */
    constructor(grid, key) {
        this.grid = grid;
        this.key = key;
    }

    load() {
        if (!this.key) return;
        let states = lx.app.storage.get('lx.ModelsGrid') || {};
        return states[this.key] || {};
    }

    commit() {
        if (!this.key) return;

        let states = lx.app.storage.get('lx.ModelsGrid') || {};

        let state = {};
        for (let fName in this.grid.fcMap) {
            let inx = this.grid.fcMap[fName];
            if (inx === null) {
                state[fName] = {v:0};
                continue;
            }
            state[fName] = {
                v: 1,
                i: inx,
                w: lx(this.grid)>>columnWrapper.at(inx).width('px')
            };
        }
        
        states[this.key] = state;
        lx.app.storage.set('lx.ModelsGrid', states);
    }
}

// @lx:context>
