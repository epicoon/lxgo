// @lx:module lx.MatrixSwapper;

/**
 * @widget lx.MatrixSwapper
 * @content-disallowed
 *
 * Events:
 * - swapped
 */
// @lx:namespace lx;
class MatrixSwapper extends lx.Box {
    // @lx:<context CLIENT:
    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Box::constructor::config),
     *     [matrixItem] {lx.Box}
     * }}
     */
    clientRender(config) {
        if (config.matrixItem) this.matrixItem = config.matrixItem;
        this.__env = null;
        this.on('mousedown', ()=>_onMouseDown(this));
    }
    // @lx:context>
}

// @lx:<context CLIENT:
function _onMouseDown(self) {
    let item = _getItem(self);
    if (!item) return;

    let config = {
        key: '__holder',
        width: item.width('px') + 'px',
        height: item.height('px') + 'px'
    };

    self.__env = {
        stream: null,
        width: null,
        height: null,
        next: null,
        holder: null,
        depthCluster: null,
        holderI: null,
        list: []
    };

    self.__env.stream = item.parent;
    self.__env.dc = item.getDepthCluster();
    self.__env.width = item.width();
    self.__env.height = item.height();
    let next = item.nextSibling();
    if (next)
        config.before = next;
    else config.parent = item.parent;

    let geom = item.getGlobalRect();
    item.setParent(lx.app.root);
    item.style('position', 'absolute');
    item.setDepthCluster(lx.DepthClusterMap.CLUSTER_URGENT);
    item.left(geom.left + 'px');
    item.top(geom.top + 'px');
    item.width(geom.width + 'px');
    item.height(geom.height + 'px');
    item.move();
    item.__sir = self;
    item.on('moveEnd', ()=>_onMoveEnd(self));
    item.on('move', ()=>_onMove(self));

    self.__env.holder = new lx.Box(config);
    _calcList(self, item.width('px'), item.height('px'));
}

function _onMove(self) {
    let item = _getItem(self);
    if (!item) return;

    let r = item.getGlobalRect(),
        xMid = r.left + 0.5 * r.width,
        yMid = r.top + 0.5 * r.height,
        boxData = null;
    for (let pare of self.__env.list) {
        let data = pare[1];
        if (xMid >= data.l && xMid <= data.r
            && yMid >= data.t && yMid <= data.b
        ) {
            boxData = data;
            break;
        }
    }
    if (!boxData || boxData.elem.key == '__holder') return;

    const stream = self.__env.stream,
        holder = self.__env.holder,
        holderI = self.__env.holderI;
    let hNext = stream.child(boxData.i + 1),
        cNext = stream.child(holderI + 1);
    if (hNext === self.__env.holder)
        hNext = boxData.elem;

    let from = self.__env.list.get(holder).i, to;
    if (hNext) {
        to = self.__env.list.get(hNext).i;
        holder.setParent({before: hNext});
    } else {
        to = stream.childrenCount() - 1;
        holder.setParent({parent: stream});
    }

    holder.width(item.width('px') + 'px');
    holder.height(item.height('px') + 'px');

    if (cNext !== boxData.elem) {
        if (boxData.elem.nextSibling() !== cNext)
            boxData.elem.setParent(cNext ? {before: cNext} : {parent: stream});
    }

    _calcList(self, item.width('px'), item.height('px'));

    const e = self.newEvent({from, to});
    self.trigger('swapped', e);
    self.__env.stream.trigger('swapped', e);
    if (item !== self) item.trigger('swapped', e);
}

function _onMoveEnd(self) {
    let item = _getItem(self);
    if (!item) return;

    item.style('position', 'relative');

    item.left(null);
    item.top(null);
    item.setParent({ before: self.__env.holder });
    item.width(self.__env.width);
    item.height(self.__env.height);
    item.setDepthCluster(self.__env.dc);

    const holderI = self.__env.holderI;

    self.__env.holder.del();
    self.__env = null;
    item.off('moveEnd');
    item.off('move');
    item.move(false);
}

function _getItem(self) {
    if (self.matrixItem) return self.matrixItem;

    let parent = self;
    while (parent) {
        if (parent.parent && (
            parent.parent.positioning().lxClassName() == 'StreamPositioningStrategy'
            || parent.parent.positioning().lxClassName() == 'GridPositioningStrategy'
        )) {
            self.matrixItem = parent;
            return self.matrixItem;
        }
        parent = parent.parent;
    }
    return null;
}

function _calcList(self, w, h) {
    self.__env.list = new Map();
    self.__env.stream.getChildren().forEach((elem, i)=>{
        if (elem.isDisplay()) {
            if (elem.key == '__holder')
                self.__env.holderI = i;
            elem.getGlobalRect();
            let rect = elem.getGlobalRect();
            self.__env.list.set(elem, {
                elem, i,
                l: rect.left,
                r: rect.left + Math.min(rect.width, w),
                t: rect.top,
                b: rect.top + Math.min(rect.height, h),
            });
        }
    });
}
// @lx:context>
