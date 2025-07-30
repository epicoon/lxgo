// @lx:namespace lx;
class RequestSynchronizer {
	constructor(callback) {
		this.keyCounter = 0;
		this.waitingList = [];
		this.callback = callback;
	}

	register(request) {
		let onLoad = request.getLoadCallback();
		let onError = request.getErrorCallback();

		request.synchronKey = this.genKey();
		this.waitingList[request.synchronKey] = {
			request,
			status: false,
			onLoad,
			onError
		};

		request.onLoad((...args)=>{
			this.ready(request);
			if (lx.isFunction(onLoad))
				onLoad.apply(request, args);
		});
		request.onError((...args)=>{
			this.ready(request);
			if (lx.isFunction(onError))
				onError.apply(request, args);
		});
	}

	send() {
		lx.app.animation.addTimer(this);
		for (let key in this.waitingList) {
			this.waitingList[key].request.send();
		}
		return this;
	}

	then(callback) {
		this.callback = callback;
	}

	stop() {
		lx.app.animation.removeTimer(this);
		this.keyCounter = 0;
		this.waitingList = [];
	}

	go() {
		if (this.allAreReady()) {
			this.stop();
			if (lx.isFunction(this.callback)) this.callback();
		}
	}

	genKey() {
		return 'k' + (this.keyCounter++);
	}

	allAreReady() {
		for (var i in this.waitingList) {
			if (this.waitingList[i].status === false)
				return false;
		}
		return true;
	}

	ready(request) {
		var info = this.waitingList[request.synchronKey];
		info.status = true;
		delete request.synchronKey;
		request.onLoad(info.onLoad);
		request.onError(info.onError);
	}
}
