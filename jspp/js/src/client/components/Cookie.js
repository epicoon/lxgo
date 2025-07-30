// @lx:namespace lx;
class Cookie extends lx.AppComponent {
	get(name) {
		var matches = document.cookie.match(new RegExp(
			"(?:^|; )" + name.replace(/([\.$?*|{}\(\)\[\]\\\/\+^])/g, '\\$1') + "=([^;]*)"
		));
		return matches ? decodeURIComponent(matches[1]) : undefined;
	}

	set(name, value, options={}) {
		var expires = options.expires;

		if (typeof expires == "number" && expires) {
			var d = new Date();
			d.setTime(d.getTime() + expires * 1000);
			expires = options.expires = d;
		}
		if (expires && expires.toUTCString) {
			options.expires = expires.toUTCString();
		}

		value = encodeURIComponent(value);

		var updatedCookie = name + "=" + value;

		for (var propName in options) {
			updatedCookie += "; " + propName;
			var propValue = options[propName];
			if (propValue !== true) {
				updatedCookie += "=" + propValue;
			}
		}

		document.cookie = updatedCookie;
	}

	remove(name) {
		this.set(name, "", {
			expires: -1
		});
	}

	getNames() {
		var reg = /(^|; *)(.+?)=/g,
			match,
			names = [];
		while (match = reg.exec(document.cookie)) names.push(match[2]);
		return names;
	}

	removeAll() {
		var names = this.getNames();
		for (var i=0, l=names.len; i<l; i++) this.remove(names[i]);
	}
}
