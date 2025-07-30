// @lx:module lx.TableManager;

// When unselect a table, the cell currently selected will be remembered and will continue to be highlighted
const DEFAULT_CELLS_FOR_SELECT = false;
// Enables auto-adding of lines when moving the cursor down from the last line
const DEFAULT_AUTO_ROW_ADDING = false;
// Allows text entry into cells
const DEFAULT_CELL_ENTER_ENABLE = true;

let __isActive = false;
let __tables = [];
let __activeTable = null;
let __activeCell = null;

/**
 * Events:
 * - selectionChange
 * - rowAdded
 */
// @lx:namespace lx;
class TableManager extends lx.Element {
    static initCss(css) {
        css.addClasses({
            'lx-TM-table': 'border: ' + css.preset.checkedMainColor + ' solid 3px !important',
            'lx-TM-row': 'background-color: ' + css.preset.checkedMainColor + ' !important',
            'lx-TM-cell': 'background-color: ' + css.preset.checkedMainColor + ' !important'
        });
    }
    
    static register(table, config = {}) {
        lx.app.cssManager.addElement(this);

        if (!(table instanceof lx.Table) || __tables.includes(table)) return;
        if (__tables.lxEmpty()) this.start();
        __tables.push(table);

        table.__interactiveInfo = {
            cellsForSelect: lx.getFirstDefined(config.cellsForSelect, DEFAULT_CELLS_FOR_SELECT),
            autoRowAdding: lx.getFirstDefined(config.autoRowAdding, DEFAULT_AUTO_ROW_ADDING),
            cellEnterEnable: lx.getFirstDefined(config.cellEnterEnable, DEFAULT_CELL_ENTER_ENABLE)
        };

        table.activeRow = function() {
            if (!this.interactiveInfo || !this.interactiveInfo.cellsForSelect) return null;
            if (this.interactiveInfo.row === undefined) return null;
            return this.interactiveInfo.row;
        }

        table.activeCell = function() {
            if (!this.interactiveInfo || !this.interactiveInfo.cellsForSelect) return null;
            if (this.interactiveInfo.cell === undefined) return null;
            return this.interactiveInfo.cell;
        }

        table.on('click', _handlerClick);
    }

    static unregister(tab) {
        if (__tables.lxEmpty()) return;
        var index = __tables.indexOf(tab);
        if (index != -1) {
            var table = __tables[index];
            delete table.__interactiveInfo;
            delete table.activeCell;
            delete table.activeRow;
            table.off('click', _handlerClick);
            __tables.splice(index, 1);
        }
        if (__tables.lxEmpty()) this.stop();
    }

    static start() {
        if (__isActive) return;
        lx.on('keydown', _handlerKeyDown);
        lx.on('keyup', _handlerKeyUp);
        lx.on('mouseup', _handlerOutclick);
        __isActive = true;
    }

    static stop() {
        if (!__isActive) return;
        this.unselect();
        lx.off('keydown', _handlerKeyDown);
        lx.off('keyup', _handlerKeyUp);
        lx.off('mouseup', _handlerOutclick);
        __isActive = false;
    }

    static unselect() {
        if (!__activeTable) return;

        __activeTable.trigger('selectionChange', __activeTable.newEvent({
            newCell: null,
            oldCell: __activeCell
        }));

        _removeClasses(__activeTable, __activeCell);
        __activeCell = null;
        __activeTable = null;
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Style
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
lx.TableManager.tableCss = 'lx-TM-table';
lx.TableManager.rowCss = 'lx-TM-row';
lx.TableManager.cellCss = 'lx-TM-cell';

function _applyClasses(tab, cell) {
    tab.addClass(lx.TableManager.tableCss);
    if (cell) cell.addClass(lx.TableManager.cellCss);
}

function _removeClasses(tab, cell) {
    tab.removeClass(lx.TableManager.tableCss);
    if (cell && !tab.__interactiveInfo.cellsForSelect)
        cell.removeClass(lx.TableManager.cellCss);
}

function _actualizeCellClass(oldCell, newCell) {
    if (oldCell != undefined) oldCell.removeClass(lx.TableManager.cellCss);
    if (newCell != undefined) newCell.addClass(lx.TableManager.cellCss);
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Handlers
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _handlerClick(event) {
    event = event || window.event;

    var target = event.target.__lx;
    if (!target) return;

    var newCell = target.lxClassName() == 'TableCell'
        ? target
        : target.ancestor((ancestor)=>ancestor.lxClassName() == 'TableCell');
    if (!newCell) return;

    if (__activeCell == newCell) {
        _enterCell(event);
        return;
    }

    var lastTab = __activeTable,
        lastCell = __activeCell,
        newTab = newCell.table();

    if (lastTab && newTab == lastTab) _actualizeCellClass(lastCell, newCell);
    else {
        if (lastTab) _removeClasses( lastTab, lastCell );
        __activeTable = newTab;
        _applyClasses( newTab, newCell );
    }

    __activeCell = newCell;
    if (newTab.__interactiveInfo.cellsForSelect) {
        var ac = newTab.activeCell();
        if (ac) ac.removeClass(lx.TableManager.cellCss);
        var coords = newCell.indexes();
        newTab.__interactiveInfo.row = coords[0];
        newTab.__interactiveInfo.col = coords[1];
    }

    event.newCell = newCell;
    event.oldCell = lastCell;
    newTab.trigger('selectionChange', event);
}

function _handlerOutclick(event) {
    if (!__activeTable) return;
    event = event || window.event;
    if ( __activeTable.containGlobalPoint(event.clientX, event.clientY) ) return;
    lx.TableManager.unselect();
}

function _handlerKeyDown(event) {
    if (__activeTable == null) return;
    event = event || window.event;
    var code = (event.charCode) ? event.charCode: event.keyCode;

    var inputOn = ( __activeCell && __activeCell.contains('input') );

    switch (code) {
        case 38: _toUp(event);    if (!inputOn) event.preventDefault(); break;
        case 40: _toDown(event);  if (!inputOn) event.preventDefault(); break;
        case 37: _toLeft(event);  if (!inputOn) event.preventDefault(); break;
        case 39: _toRight(event); if (!inputOn) event.preventDefault(); break;
    }
}

function _handlerKeyUp(event) {
    if (__activeTable == null) return;
    event = event || window.event;
    var code = (event.charCode) ? event.charCode: event.keyCode;
    if (code == 13) _enterCell(event);
    else if (code == 27) {
        let cell = __activeCell;

        console.log(cell);

        if (cell.isEditing()) cell.trigger('blur');
    }
}

function _enterCell(event) {
    let tab = __activeTable;
    if (!tab || !tab.__interactiveInfo.cellEnterEnable) return;

    let cell = __activeCell;
    if (cell && !cell.isEditable()) {
        cell.setEditable(true);
        cell.on('blur', ()=>cell.setEditable(false));
        cell.edit();
    }
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Move
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _toUp(event) {
    var cell = __activeCell;
    if ( cell.contains('input') ) return;

    var tab = __activeTable,
        coords = cell.indexes(),
        rowNum = coords[0],
        colNum = coords[1];

    if ( !rowNum ) return;

    var newRow = tab.row(rowNum - 1),
        newCell = tab.cell(rowNum - 1, colNum);

    if (tab.__interactiveInfo.cellsForSelect)
        tab.__interactiveInfo.row = rowNum - 1;
    __activeCell = newCell;

    _actualizeCellClass(cell, newCell);

    var scr = tab.getDomElem().scrollTop,
        rT = newRow.getDomElem().offsetTop;
    if ( rT < scr ) tab.getDomElem().scrollTop = rT;

    event = event || tab.newEvent();
    event.newCell = newCell;
    event.oldCell = cell;
    tab.trigger('selectionChange', event);
}

function _toDown(event) {
    var cell = __activeCell;
    if ( cell.contains('input') ) return;

    event = event || __activeTable.newEvent();

    var tab = __activeTable,
        coords = cell.indexes(),
        rowNum = coords[0],
        colNum = coords[1];

    if ( tab.rowsCount() == rowNum + 1 ) {
        if (tab.__interactiveInfo.autoRowAdding) {
            tab.addRow();
            tab.trigger('rowAdded', event);
        } else return;
    }

    var newRow = tab.row( rowNum + 1 ),
        newCell = tab.cell(rowNum + 1, colNum);

    if (tab.__interactiveInfo.cellsForSelect)
        tab.__interactiveInfo.row = rowNum + 1;
    __activeCell = newCell;

    _actualizeCellClass(cell, newCell);

    var scr = tab.getDomElem().scrollTop,
        h = tab.getDomElem().offsetHeight,
        rT = newRow.getDomElem().offsetTop,
        rH = newRow.getDomElem().offsetHeight;
    if ( rT + rH > scr + h ) tab.getDomElem().scrollTop = rT + rH - h;

    event.newCell = newCell;
    event.oldCell = cell;
    tab.trigger('selectionChange', event);
}

function _toLeft(event) {
    var cell = __activeCell;
    if ( cell.contains('input') ) return;

    var tab = __activeTable,
        coords = cell.indexes(),
        rowNum = coords[0],
        colNum = coords[1];

    if ( !colNum ) return;

    var newCell = tab.cell(rowNum, colNum - 1);

    if (tab.__interactiveInfo.cellsForSelect)
        tab.__interactiveInfo.col = colNum - 1;
    __activeCell = newCell;

    _actualizeCellClass(cell, newCell);

    var scr = tab.getDomElem().scrollLeft,
        rL = newCell.getDomElem().offsetLeft;
    if ( rL < scr ) tab.getDomElem().scrollLeft = rL;

    event = event || tab.newEvent();
    event.newCell = newCell;
    event.oldCell = cell;
    tab.trigger('selectionChange', event);
}

function _toRight(event) {
    var cell = __activeCell;
    if ( cell.contains('input') ) return;

    var tab = __activeTable,
        coords = cell.indexes(),
        rowNum = coords[0],
        colNum = coords[1];

    if ( tab.colsCount(rowNum) == colNum + 1 ) return;

    var newCell = tab.cell(rowNum, colNum + 1);

    if (tab.__interactiveInfo.cellsForSelect)
        tab.__interactiveInfo.col = colNum + 1;
    __activeCell = newCell;

    _actualizeCellClass(cell, newCell);

    var scr = tab.getDomElem().scrollLeft,
        w = tab.getDomElem().offsetWidth,
        rL = newCell.getDomElem().offsetLeft,
        rW = newCell.getDomElem().offsetWidth;
    if ( rL + rW > scr + w ) tab.getDomElem().scrollLeft = rL + rW - w;

    event = event || tab.newEvent();
    event.newCell = newCell;
    event.oldCell = cell;
    tab.trigger('selectionChange', event);
}
