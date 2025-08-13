// @lx:namespace lx;
class Date extends lx.Object {
	constructor(...args) {
		super();
		this.reset(...args);
	}

	static __afterDefinition() {
		super.__afterDefinition();
		this.delegateMethods({
			jsDate: lx.globalContext.Date
		});
	}

	reset(...args) {
		this.jsDate= (args.length == 1 && args[0] instanceof Date)
			? args[0]
			: new lx.globalContext.Date(...args);
	}

	getYear(format = null) {
		if (format == 'y') return this.jsDate.getFullYear() % 100;
		return this.jsDate.getFullYear();
	}

	getMonth(format = null) {
		if (format === null) return this.jsDate.getMonth();
		if (format != 'm') return '' + (this.jsDate.getMonth() + 1);
		let month = this.jsDate.getMonth() + 1;
		return ((month < 10) ? '0' : '') + month;
	}

	getDate(format = null) {
		if (format === null) return this.jsDate.getDate();
		if (format != 'd') return '' + this.jsDate.getDate();
		let date = this.jsDate.getDate();
		return ((date < 10) ? '0' : '') + date;
	}

	getMaxDate() {
		return lx.self(getMaxDate(this.getYear(), this.getMonth()));
	}

	static getMaxDate(y, m) {
		if (m == 1) return y%4 || (!(y%100) && y%400 ) ? 28 : 29;
		return m===3 || m===5 || m===8 || m===10 ? 30 : 31;
	}

	shiftDay(count) {
		this.jsDate.setDate(this.jsDate.getDate() + count);
		return this;
	}

	shiftMonth(count) {
		this.jsDate.setMonth(this.jsDate.getMonth() + count);
		return this;
	}

	shiftYear(count) {
		this.jsDate.setFullYear(this.jsDate.getFullYear() + count);
		return this;
	}

	isLater(date) {
		return this.valueOf() > date.valueOf();
	}

	getDaysBetween(date) {
		let oneDay = 1000*60*60*24;
		let date1, date2;
		if (this.isLater(date)) {
			date2 = this;
			date1 = date;
		} else {
			date2 = date;
			date1 = this;
		}
		return Math.ceil((date2.getTime() - date1.getTime()) / oneDay);
	}

	format(format = 'Y-m-d') {
		let result = '';
		for (let i=0, l=format.length; i<l; i++) {
			let letter = format[i];
			switch (letter) {
				case 'Y':
				case 'y':
					result += this.getYear(letter)
					break;
				case 'm':
					result += this.getMonth(letter)
					break;
				case 'd':
					result += this.getDate(letter)
					break;
				case 'h':
					result += this.jsDate.getHours()
					break;
				case 'i':
					result += this.jsDate.getMinutes()
					break;
				case 's':
					result += this.jsDate.getSeconds()
					break;
				case 'l':
					result += this.jsDate.getMilliseconds()
					break;
				default:
					result += letter;
			}
		}
		return result;
	}
}
