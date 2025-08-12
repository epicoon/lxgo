let __moved = false,
	__movedDelta = { x: 0, y: 0 },
	__movedElement = null;

// @lx:namespace lx;
class DragNDrop extends lx.AppComponent {
	move(event) {
		event = event || window.event;
		lx.preventDefault(event);

		__moved = true;
		__movedElement = this;

		var X = event.clientX || event.changedTouches[0].clientX,
			Y = event.clientY || event.changedTouches[0].clientY;

		var el = (this.moveParams.parentMove || this.moveParams.parentResize) ? this.parent : this;
		el.emerge();

		delete el.geom.bpg;
		delete el.geom.bpv;

		if (this.moveParams.parentResize) {
			var p = this.parent;
			__movedDelta.x = X - p.left('px') - p.width('px');
			__movedDelta.y = Y - p.top('px') - p.height('px');
			this.trigger('moveBegin', event);
			this.parent.trigger('resizeBegin', event);
			return;
		}

		__movedDelta.x = X - el.left('px');
		__movedDelta.y = Y - el.top('px');

		if (this.moveParams.parentMove) this.trigger('moveBegin', event);
		el.trigger('moveBegin', event);
	}

	useElementMoving(bool = true) {
		var method = bool ? 'on' : 'off';
		lx[method]('mousemove', __watchForMove);
		lx[method]('mouseup', __resetMovedElement);
		lx[method]('touchmove', __watchForMove);
		lx[method]('touchend', __resetMovedElement);
	}
}

function __resetMovedElement(event) {
	if (__movedElement == null) return;

	var el = __movedElement;
	__moved = false;
	__movedElement = null;

	el.trigger('moveEnd', event);
	if (el.moveParams.parentResize && el.parent) el.parent.trigger('resizeEnd', event);
}

function __watchForMove(event) {
	if (!__moved) return;
	if (__movedElement == null) return;

	if (__movedElement.moveParams.locked) {
		__moved = false;
		__movedElement = null;
		return;
	}

	event = event || window.event;

	var el = __movedElement,
		info = el.moveParams;
	
	var X, Y;
	if (event.clientX) X = event.clientX;
	else if (event.changedTouches) X = event.changedTouches[0].clientX;
	if (event.clientY) Y = event.clientY;
	else if (event.changedTouches) Y = event.changedTouches[0].clientY;

	var newPos = {
		x: X - __movedDelta.x,
		y: Y - __movedDelta.y
	};

	if (info.parentMove) newPos = __limitPosition(el.parent, newPos, info);
	else if (info.parentResize) newPos = __limitPositionForResize(el, newPos, info);
	else newPos = __limitPosition(el, newPos, info);

	el.trigger('beforeMove', event, newPos);

	if (info.parentResize) {
		var p = el.parent;
		if (info.xMove) p.width( newPos.x - p.left('px') + 'px' );
		if (info.yMove) p.height( newPos.y - p.top('px') + 'px' );
		el.trigger('move', event);
		p.checkResize(event);
		return;
	}

	var movedEl = (info.parentMove) ? el.parent : el;
	if (info.xMove) movedEl.left( newPos.x + 'px' );
	if (info.yMove) movedEl.top( newPos.y + 'px' );

	if (info.parentMove) el.trigger('move', event);
	movedEl.trigger('move', event);
}

function __limitPositionForResize(el, newPos, info) {
	var p = el.parent, pp = p.parent;
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

function __limitPosition(el, newPos, info) {
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
