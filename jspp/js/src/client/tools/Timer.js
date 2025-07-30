// @lx:namespace lx;
class Timer {
	/**
	 * @param config {Number|Array<Number>|Object {
	 *     period {Number}
	 * }}
	 */
	constructor(config=0) {
		if (lx.isNumber(config) || lx.isArray(config)) config = {period: config};

		// If array, periods will alternate
		// Milliseconds
		this.periodDuration = config.period;

		// For every frame update
		// this.actions - if array, functions will be called sequentially
		this.actions = config.action || config.actions || null;

		this.inAction = false;
		this.startTime = (new Date).getTime();
		this.countPeriods = lx.getFirstDefined(config.countPeriods, true);
		this.periodCounter = 0;
		this.periodIndex = 0;
		this.actionIndex = 0;


		this.task = null;
		this._action = function(){};
		this._iteration = function(){};
	}

	/**
	 * For every frame update
	 */
	whileCycle(func) { this._action = func; }

	/**
	 * For period ends
	 */
	onCycleEnd(func) { this._iteration = func; }

	/**
	 * Are implemented in parallel: method ._action() and field .actions
	 * - the method can be initialized using the method [[whileCycle(func)]]
	 * - the field can be initialized with an array of functions that will run sequentially
	 * - the method and functions from the array field can work in parallel (the method in the iteration will work first)
	 */
	setAction(actions) {
		this.actions = actions;
	}

	start() {
		if (this.inAction) return;
		this.startTime = (new Date).getTime();
		lx.app.animation.addTimer(this);
		this.inAction = true;
	}

	syncStart(queueName = null) {
		this.task = new lx.Task(queueName);
		this.task.onChangeStatus(()=>{
			if (this.task.isPending())
				this.startTime = (new Date).getTime();
		});
		this.start();
	}

	/**
	 * The period counter is not reset when stopped! If you need to reset it, use the resetCounter() method
	 */
	stop() {
		lx.app.animation.removeTimer(this);
		this.startTime = 0;
		this.inAction = false;
		this.periodIndex = 0;
		this.actionIndex = 0;
		if (this.task) {
			this.task.setCompleted();
			this.task = null;
		}
	}

	/**
	 * Resetting the calculated time within a period
	 */
	resetTime() {
		this.startTime = (new Date).getTime();
	}

	/**
	 * Reset period counter
	 */
	resetCounter() {
		this.periodCounter = 0;
	}
	
	getCounter() {
		return this.periodCounter;
	}

	/**
	 * Relative time shift from the beginning of the period to the current moment - value from 0 to 1
	 */
	shift() {
		var time = (new Date).getTime(),
			delta = time - this.startTime,
			k = delta / __periodDuration(this);
		if (k > 1) k = 1;
		return k;
	}

	/**
	 * Checking if the current period has ended
	 */
	isCycleEnd() {
		// If the period is not set - constant triggering
		if (!this.periodDuration) return true;

		var time = (new Date).getTime();
		if (time - this.startTime >= __periodDuration(this)) {
			this.startTime = time;
			return true;
		}
		return false;
	}

	/**
	 * Used by the system - called every time the frame is updated
	 */
	go() {
		if (this.task) {
			if (!this.task.isPending()) return;
		}

		this._action.call(this);
		var action = __action(this);
		if (lx.isFunction(action)) action.call(this);
		if (!this.inAction) return;

		if (this.isCycleEnd()) {
			if (this.countPeriods) this.periodCounter++;
			this._iteration.call(this);

			if (this.periodDuration && lx.isArray(this.periodDuration)) {
				this.periodIndex++;
				if (this.periodIndex == this.periodDuration.length) this.periodIndex = 0;
			}

			if (this.actions && lx.isArray(this.actions)) {
				this.actionIndex++;
				if (this.actionIndex == this.actions.length) this.actionIndex = 0;
			}
		}
	}
}

function __periodDuration(self) {
	if (self.periodDuration && lx.isArray(self.periodDuration))
		return self.periodDuration[self.periodIndex];
	return self.periodDuration;
}

function __action(self) {
	if (self.actions && lx.isArray(self.actions))
		return self.actions[self.actionIndex];
	return self.actions;
}
