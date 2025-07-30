let _map = {};

// @lx:namespace lx;
class DepthClusterMap {
	// @lx:const CLUSTER_DEEP = 0;
	// @lx:const CLUSTER_PRE_MIDDLE = 1;
	// @lx:const CLUSTER_MIDDLE = 2;
	// @lx:const CLUSTER_PRE_FRONT = 3;
	// @lx:const CLUSTER_FRONT = 4;
	// @lx:const CLUSTER_PRE_OVER = 5;
	// @lx:const CLUSTER_OVER = 6;
	// @lx:const CLUSTER_URGENT = 7;

	static calculateZIndex(cluster) {
		return cluster * this.getClusterSize();
	}

	static getClusterSize() {
		return 1000;
	}

	// @lx:<context CLIENT:
	static bringToFront(el) {
		var zShift = _zIndex(el);
		var key = 's' + zShift;
		if (_map[key] === undefined) _map[key] = [];

		if (el.__frontIndex !== undefined) {
			if (el.__frontIndex == _map[key].len - 1) return;
			_removeFromFrontMap(el);
		}

		var map = _map[key];

		if (el.getDomElem() && el.getDomElem().offsetParent) {
			el.__frontIndex = map.len;
			el.style('z-index', el.__frontIndex + zShift);
			map.push(el);
		}
	}

	static checkFrontMap() {
		var shown = 0,
			newFrontMap = {};
		for (var key in _map) {
			var map = _map[key];
			var newMap = [];
			for (var i = 0, l = map.len; i < l; i++) {
				if (map[i].getDomElem() && map[i].getDomElem().offsetParent) {
					var elem = map[i];
					elem.__frontIndex = shown;
					elem.style('z-index', map[i].__frontIndex + _zIndex(elem));
					newMap.push(elem);
					shown++;
				}
			}
			newFrontMap[key] = newMap;
		}

		_map = newFrontMap;
	}
	// @lx:context>
}

// @lx:<context CLIENT:
function _removeFromFrontMap(el) {
	if (el.__frontIndex === undefined) return;

	var zShift = _zIndex(el);
	var key = 's' + zShift;
	if (_map[key] === undefined) _map[key] = [];
	var map = _map[key];

	for (var i = el.__frontIndex + 1, l = map.len; i < l; i++) {
		map[i].__frontIndex = i - 1;
		map[i].style('z-index', i - 1 + zShift);
	}
	map.splice(el.__frontIndex, 1);
}

function _zIndex(el) {
	return lx.DepthClusterMap.calculateZIndex(el.getDepthCluster());		
}
// @lx:context>
