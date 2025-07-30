// @lx:module lx.Table;

/**
 * @widget lx.Table
 * @content-disallowed
 * 
 * CSS classes:
 * - lx-Table
 * - lx-Table-row
 * - lx-Table-cell
 */
// @lx:namespace lx;
class Table extends lx.Box  {
	static initCss(css) {
		css.addClass('lx-Table', {
			borderRadius: css.preset.borderRadius,
			color: css.preset.textColor
		});
		css.addClass('lx-Table-row', {
			overflow: 'hidden',
		}, {
			'first-child': 'border: 0px',
			'nth-child(2n)': 'background-color: ' + css.preset.bodyBackgroundColor,
			'nth-child(2n+1)': 'background-color: ' + css.preset.altBodyBackgroundColor
		});
		css.addClass('lx-Table-cell', {
			height: '100%',
		}, {
			'last-child': 'border: 0px'
		});
	}

	/**
	 * @widget-init
	 *
	 * @param [config] {Object: {
	 *     #merge(lx.Rect::constructor::config),
	 *     [rows = 0] {Number},
	 *     [cols = 0] {Number},
	 *     [rowHeight] {Number|String},
	 *     [borders = false] {Boolean|String|Tuple: [
	 *         (: size in pixels :) {Number},
	 *         (: type :) {String},
	 *         (: color :) {lx.Color|String|Tuple:[(:R:){Number}, (:G:){Number}, (:B:){Number}]}
	 *     ]},
	 *     [indents] {Object: #schema(lx.IndentData::constructor::config)}
	 * }}
	 */
	render(config) {
		super.render(config);
		this.addClass('lx-Table');

		let rows = config.rows || 0;
		this.cols = config.cols || 0;

		this.indents = new lx.IndentData(config.indents || {});
		let indentData = this.indents.get(),
			rowStreamConfig = {};
		if (config.rowHeight === undefined)
			rowStreamConfig.type = lx.StreamPositioningStrategy.TYPE_PROPORTIONAL;
		else rowStreamConfig.height = config.rowHeight;
		if (indentData.stepY)         rowStreamConfig.stepY         = indentData.stepY;
		if (indentData.paddingTop)    rowStreamConfig.paddingTop    = indentData.paddingTop;
		if (indentData.paddingBottom) rowStreamConfig.paddingBottom = indentData.paddingBottom;
		this.stream(rowStreamConfig);

		if (!rows || !this.cols) return;
		this.insertRows(-1, rows);

		const borders = config.borders || false;
		//TODO if (borders)
	}

	// @lx:<context SERVER:
	beforePack() {
		this.indents = this.indents.pack();
	}
	// @lx:context>

	// @lx:<context CLIENT:
	postUnpack(config) {
		super.postUnpack(config);
		this.indents = lx.IndentData.unpackOrNull(this.indents);
	}
	// @lx:context>

	row(row) {
		if (!lx(this)>r) return null;
		if (lx(this)>r instanceof lx.TableRow) return lx(this)>r;
		if (!lx(this)>r[row]) return null;
		return lx(this)>r[row];
	}

	cell(row, col) {
		let r = this.row(row);
		if (!r) return null;
		return r.cell(col);
	}

	rowsCount() {
		if (!lx(this)>r) return 0;
		if (lx.isArray(lx(this)>r))
			return lx(this)>r.len;
		return 1;
	}

	colsCount(row) {
		if (row === undefined) return this.cols;
		let r = this.row(row);
		if (r) return r.cellsCount();
		return 0;
	}

	rows(r0=0, r1=null) {
		let c = new lx.Collection(),
			rows = this.rowsCount();
		if (!rows) return c;
		if (r1 === null || r1 >= rows) r1 = rows - 1;

		if (r0 == 0 && r1 == rows - 1) return c.add(lx(this)>r);

		for (let i=r0; i<=r1; i++) c.add( this.row(i) );
		return c;
	}

	cells(r0=0, c0=0, r1=null, c1=null) {
		let c = new lx.Collection(),
			rows = this.rowsCount();

		if (r1 === null || r1 >= rows) r1 = rows - 1;

		for (let i=r0; i<=r1; i++) {
			let r = this.row(i),
				cols = this.colsCount(i);
			if (c1 === null || c1 >= cols) c1 = cols - 1;

			if (c0 == 0 && c1 == cols - 1) c.add(lx(r)>c);
			else {
				for (let j=c0; j<=c1; j++)
					c.add( this.cell(i, j) );
			}
		}

		return c;
	}

	eachRow(func) {
		this.rows().forEach(func);
	}

	/**
	 * @param {Function} func - callback function
	 * @param {Boolean} transpon - false as default, rows in priority while filling
	 * @param {Number} r0 - row to start
	 * @param {Number} c0 - column to start
	 * @param {Number} r1 - row to finish
	 * @param {Number} c1 - column to finish
	 */
	eachCell(func, transpon = false, r0 = 0, c0 = 0, r1 = null, c1 = null) {
		let rows = this.rowsCount(),
			cols = this.colsCount(),
			counter = 0;
		if (r1 === null || r1 >= rows) r1 = rows - 1;
		if (c1 === null || c1 >= cols) c1 = cols - 1;

		if (transpon) {
			for (let j=c0; j<=c1; j++)
				for (let i=r0; i<=r1; i++)
					if (func(this.cell(i, j), i, j, counter++) === false)
						return this;
			return this;
		}

		for (let i=r0; i<=r1; i++)
			for (let j=c0; j<=c1; j++)
				if (func(this.cell(i, j), i, j, counter++) === false)
					return this;

		return this;
	}

	/**
	 * @param {Array} content - line or two-dimensional array
	 * @param {Boolean} transpon - false as default, rows in priority while filling
	 * @param {Number} r0 - row to start filling
	 * @param {Number} c0 - column to start filling
	 */
	setContent(content, transpon = false, r0 = 0, c0 = 0) {
		let r1, c1;

		if (!lx.isArray(content[0])) content = [content];

		if (transpon) {
			r1 = r0 + content[0].len;
			c1 = c0 + content.len - 1;
		} else {
			r1 = r0 + content.len - 1;
			c1 = c0 + content[0].len;
		}

		if (r1 > this.rowsCount()) this.setRowsCount(r1);

		this.eachCell((cell, r, c)=> {
			cell.text( transpon ? content[c - c0][r - r0] : content[r - r0][c - c0] );
		}, transpon, r0, c0, r1, c1);
	}

	resetContent(content, transpon=false, r0=0, c0=0) {
		this.clear();
		if (content.len)
			this.setContent(content, transpon, r0, c0);
	}

	insertRows(next, amt) {
		let cols = this.colsCount(),
			row = this.row(next),
			config = {key: 'r'};
		if (row) config.before = row;
		else config.parent = this;

		config.css = 'lx-Table-row';
		config.cols = cols;
		let c = lx.TableRow.construct(amt, config);
		return c;
	}

	insertRow(nextIndex) {
		this.insertRows(nextIndex, 1);
	}

	setRowsCount(rows) {
		if (rows == this.rowsCount()) return this;

		if (rows < this.rowsCount()) {
			this.del('r', rows, this.rowsCount() - rows);
			return;
		}

		return this.insertRows(-1, rows - this.rowsCount());
	}

	addRow() {
		this.addRows(1);
	}

	addRows(amt) {
		return this.setRowsCount( this.rowsCount() + amt );
	}

	delRow(num) {
		this.delRows(num, 1);
	}

	delRows(num, amt) {
		this.del('r', num, amt);
		return this;
	}

	setRowsHeight(height) {
		if (this.positioningStrategy.type == lx.StreamPositioningStrategy.TYPE_PROPORTIONAL) return this;
		this.positioningStrategy.rowDefaultHeight = height;
		this.rows().forEach(a=>a.height(height));
		return this;
	}

	setColsCount(count) {
		if (count == this.colsCount()) return;

		if (count < this.colsCount()) {
			this.eachRow((r)=>r.del('c', count, r.cellsCount() - count));
			this.colsWidths = this.row(0).style('grid-template-columns')
			this.cols = count;
			return;
		}

		let c = new lx.Collection();
		this.eachRow((r)=>c.add(r.insertCells(-1, count - r.cellsCount())));
		this.colsWidths = this.row(0).style('grid-template-columns')
		this.cols = count;
		return c;
	}

	setColWidth(col, width) {
		if (!this.rowsCount()) {
			//TODO - все равно запоминать!
			return;
		}

		this.eachRow(r=>r.cell(col).width(width));
		this.colsWidths = this.row(0).style('grid-template-columns')
	}
}

//======================================================================================================================
// @lx:namespace lx;
class TableRow extends lx.Box {
	render(config) {
		let colConfig = {direction: lx.HORIZONTAL},
			table = this.parent,
			indentData = table.indents.get();
		if (indentData) {
			if (indentData.stepX) colConfig.stepX = indentData.stepX;
			if (indentData.paddingLeft)   colConfig.paddingLeft  = indentData.paddingLeft;
			if (indentData.paddingRight)  colConfig.paddingRight = indentData.paddingRight;
		}

		if (colConfig.minWidth === undefined) colConfig.minWidth = '5px';

		this.streamProportional(colConfig);
		lx.TableCell.construct(config.cols, {parent: this, key: 'c', css: 'lx-Table-cell'});
		if (table.colsWidths) this.style('grid-template-columns', table.colsWidths)
	}

	table() {
		return this.parent;
	}

	cellsCount() {
		if (!lx(this)>c) return 0;
		if (lx.isArray(lx(this)>c))
			return lx(this)>c.len;
		return 1;
	}

	cell(num) {
		if (!lx(this)>c) return null;
		if (lx(this)>c instanceof lx.TableCell) return lx(this)>c;
		if (!lx(this)>c[num]) return null;
		return lx(this)>c[num];
	}

	cells() {
		return new lx.Collection(lx(this)>c);
	}

	insertCells(nextIndex, count) {
		let cell = this.cell(nextIndex),
			config = {key: 'c'};
		if (cell) config.before = cell;
		else config.parent = this;

		config.css = 'lx-Table-cell';
		let c = lx.TableCell.construct(count, config);
		return c;
	}
}

//======================================================================================================================
// @lx:namespace lx;
class TableCell extends lx.Box {
	table() {
		return this.parent.parent;
	}

	row() {
		return this.parent;
	}

	indexes() {
		return [
			this.parent.index || 0,
			this.index || 0
		];
	}

	rowIndex() {
		return this.parent.index;
	}
}
