let __timers = null,
	__actions = [];

// @lx:namespace lx;
class Animation extends lx.AppComponent {
	addAction(f, ctx) {
		__actions.push({ func: f, context: ctx });
	}

	resetActions() {
		__actions = [];
	}

	addTimer(timer) {
		if (!__timers) return;
		__timers.add(timer);
	}

	removeTimer(timer) {
		if (!__timers) return;
		__timers.remove(timer);
	}

	useTimers(bool) {
		if (!bool) {
			__timers = null;
			return;
		}

		__timers = {
			data: [],
			add: function(one) {
				this.data.lxPushUnique(one);
			},
			remove: function(one) {
				this.data.lxRemove(one);
			},
			go: function() {
				for (var i=0; i<this.data.length; i++) {
					if (this.data[i].go !== undefined)
						this.data[i].go();
				}
			}
		};
	}

	useAnimation() {
		(function animate() {
			requestAnimationFrame(animate);
			__doActions();
		})();
	}
}

function __doActions() {
	if (__timers !== null) __timers.go();

	for (var i=0, l=__actions.length; i<l; i++) {
		var f = __actions[i].func,
			ctx = __actions[i].context;
		f.call(ctx);
	}
}
