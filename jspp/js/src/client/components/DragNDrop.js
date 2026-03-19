let _moved = false,
	_movedDelta = { x: 0, y: 0 },
	_movedElement = null;

// @lx:namespace lx;
class DragNDrop extends lx.AppComponent {
	resetDelta(elem) {
		let X = lx.app.mouse.x,
			Y = lx.app.mouse.y,
			el = _getElem(elem);

		if (elem.moveParams.parentResize) {
			_movedDelta.x = X - el.left('px') - el.width('px');
			_movedDelta.y = Y - el.top('px') - el.height('px');
			return;
		}

		_movedDelta.x = X - el.left('px');
		_movedDelta.y = Y - el.top('px');
	}

	move(event) {
		event = event || window.event;
		lx.preventDefault(event);

		_moved = true;
		_movedElement = this;

		let X = event.clientX || event.changedTouches[0].clientX,
			Y = event.clientY || event.changedTouches[0].clientY,
			el = _getElem(this);
		el.emerge();

		delete el.geom.bpg;
		delete el.geom.bpv;

		if (this.moveParams.parentResize) {
			_movedDelta.x = X - el.left('px') - el.width('px');
			_movedDelta.y = Y - el.top('px') - el.height('px');
			this.trigger('moveBegin', event);
			el.trigger('resizeBegin', event);
			return;
		}

		_movedDelta.x = X - el.left('px');
		_movedDelta.y = Y - el.top('px');

		if (this.moveParams.parentMove) this.trigger('moveBegin', event);
		el.trigger('moveBegin', event);
	}

	useElementMoving(bool = true) {
		let method = bool ? 'on' : 'off';
		lx[method]('mousemove', _watchForMove);
		lx[method]('mouseup', _resetMovedElement);
		lx[method]('touchmove', _watchForMove);
		lx[method]('touchend', _resetMovedElement);
	}
}

function _resetMovedElement(event) {
	if (_movedElement == null) return;

	var el = _movedElement;
	_moved = false;
	_movedElement = null;

	el.trigger('moveEnd', event);
	if (el.moveParams.parentResize && el.parent) el.parent.trigger('resizeEnd', event);
}

function _watchForMove(event) {
	if (!_moved) return;
	if (_movedElement == null) return;

	if (_movedElement.moveParams.locked) {
		_moved = false;
		_movedElement = null;
		return;
	}

	event = event || window.event;

	let el = _movedElement,
		info = el.moveParams,
		X, Y;

		if (event.clientX) X = event.clientX;
	else if (event.changedTouches) X = event.changedTouches[0].clientX;
	if (event.clientY) Y = event.clientY;
	else if (event.changedTouches) Y = event.changedTouches[0].clientY;

	let newPos = {
		x: X - _movedDelta.x,
		y: Y - _movedDelta.y
	};

	if (info.parentMove) newPos = _limitPosition(_getElem(el), newPos, info);
	else if (info.parentResize) newPos = _limitPositionForResize(el, newPos, info);
	else newPos = _limitPosition(el, newPos, info);

	el.trigger('beforeMove', event, newPos);

	if (info.parentResize) {
		var p = _getElem(el);
		if (info.xMove) p.width( newPos.x - p.left('px') + 'px' );
		if (info.yMove) p.height( newPos.y - p.top('px') + 'px' );
		el.trigger('move', event);
		p.checkResize(event);
		return;
	}

	let movedEl = _getElem(el);
	if (info.xMove) movedEl.left( newPos.x + 'px' );
	if (info.yMove) movedEl.top( newPos.y + 'px' );

	if (info.parentMove) el.trigger('move', event);
	movedEl.trigger('move', event);
}

function _limitPositionForResize(el, newPos, info) {
	var p = _getElem(el), pp = p.parent;
	if (info.xMove) {
		if (info.moveStep > 1) newPos.x = Math.floor( newPos.x / info.moveStep ) * info.moveStep;
		if (info.xLimit) {
			if (newPos.x > pp.width('px')) newPos.x = pp.width('px');
			if (newPos.x < 0) newPos.x = 0;
		}
	}
	if (info.yMove) {
		if (info.moveStep > 1) newPos.y = Math.floor( newPos.y / info.moveStep ) * info.moveStep;
		if (info.yLimit) {
			if (newPos.y > pp.height('px')) newPos.y = pp.height('px');
			if (newPos.y < 0) newPos.y = 0;
		}
	}
	return newPos;
}

function _limitPosition(el, newPos, info) {
	if (info.xMove) {
		if (info.moveStep > 1) newPos.x = Math.floor( newPos.x / info.moveStep ) * info.moveStep;
		if (info.xLimit) {
			let w = el.width('px'),
				pW = el.parent.width('px');
			if (w <= pW) {
				if (newPos.x + w > pW) newPos.x = pW - w;
				if (newPos.x < 0) newPos.x = 0;
			} else {
				if (newPos.x > 0) newPos.x = 0;
				if (newPos.x + w < pW) newPos.x = pW - w;
			}
		}
	} else newPos.x = el.left('px');
	if (info.yMove) {
		if (info.moveStep > 1) newPos.y = Math.floor( newPos.y / info.moveStep ) * info.moveStep;
		if (info.yLimit) {
			let h = el.height('px'),
				pH = el.parent.height('px');
			if (h <= pH) {
				if (newPos.y + h > pH) newPos.y = pH - h;
				if (newPos.y < 0) newPos.y = 0;
			} else {
				if (newPos.y > 0) newPos.y = 0;
				if (newPos.y + h < pH) newPos.y = pH - h;
			}
		}
	} else newPos.y = el.top('px');
	return newPos;
}

function _getElem(el) {
	const info = el.moveParams;
	if (info.parentMove && info.parentResize)
		throw "Wrong widget movement settings: parentMove and parentResize can not be set at the same time";

	if (!info.parentMove && !info.parentResize)
		return el;

	if (info.parentMove === true || info.parentResize === true)
		return el.parent;

	return info.parentMove || info.parentResize;
}
