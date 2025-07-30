// @lx:module lx.JointMover;

//TODO test when created on server

/**
 * @widget lx.JointMover
 * @content-disallowed
 */
// @lx:namespace lx;
class JointMover extends lx.Rect {
    // @lx:const DEFAULT_SIZE = '6px';

    modifyConfigBeforeApply(config) {
        if (config.top) {
            config.geom = [
                0,
                config.top,
                100,
                config.size || config.height || lx(STATIC).DEFAULT_SIZE
            ];
            config.direction = lx.VERTICAL;
            delete config.top;
        } else if (config.bottom) {
            config.geom = [
                0,
                null,
                100,
                config.size || config.height || lx(STATIC).DEFAULT_SIZE,
                null,
                config.bottom
            ];
            config.direction = lx.VERTICAL;
            delete config.bottom;
        } else if (config.left) {
            config.geom = [
                config.left,
                0,
                config.size || config.width || lx(STATIC).DEFAULT_SIZE,
                100
            ];
            config.direction = lx.HORIZONTAL;
            delete config.left;
        } else if (config.right) {
            config.geom = [
                null,
                0,
                config.size || config.width || lx(STATIC).DEFAULT_SIZE,
                100,
                config.right
            ];
            config.direction = lx.HORIZONTAL;
            delete config.right;
        }
        return config;
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [direction = null] {Number&Enum(lx.VERTICAL, lx.HORIZONTAL)}
     *     [limit = 20] {Number} (: number of pixels to stop resizing :)
     * }}
     */
    render(config) {
        super.render(config);

        this.direction = config.direction || null;
        this.limit = config.limit || 20;
        if (this.direction) {
            if (this.direction == lx.VERTICAL) this.style('cursor', 'ns-resize');
            else this.style('cursor', 'ew-resize');
        }
        this.move();
        this.on('move', function() {
            let prevLim = this.getPrevLimit(),
                nextLim = this.getNextLimit();
            if (this.getDirection() == lx.VERTICAL) {
                if (this.top('px') < prevLim)
                    this.top(prevLim + 'px');
                if (this.top('px') + this.height('px') > nextLim)
                    this.top(nextLim - this.height('px') + 'px');
            } else {
                if (this.left('px') < prevLim)
                    this.left(prevLim + 'px');
                if (this.left('px') + this.width('px') > nextLim)
                    this.left(nextLim - this.width('px') + 'px');
            }
            this.actualize();
        });

        let context = this;
        this.parent.on('afterAddChild', function(e) {
            context.check(e.child);
        });
    }

    clientRender(config) {
        super.clientRender(config);
        this.actualize();

        this.parent.map('%');
        this.parent.on('resize', ()=>this.actualize());
        this.displayOnce(()=>{
            let next = this.nextSibling();
            if (!next) return;
            let context = this;
            next.displayOnce(()=>context.actualize());
        });
    }

    getDirection() {
        if (this.direction) return this.direction;
        if (!this.width('px') || !this.height('px')) return;

        this.direction = this.width('px') > this.height('px')
            ? lx.VERTICAL
            : lx.HORIZONTAL;
        if (this.direction == lx.VERTICAL) this.style('cursor', 'ns-resize');
        else this.style('cursor', 'ew-resize');
        return this.direction;
    }

    check(elem) {
        if (this.width('px') === null) return;
        if (elem === this.prevSibling())
            this.actualizePrev(elem);
        else if (elem === this.nextSibling())
            this.actualizeNext(elem);
    }

    actualize() {
        if (this.width('px') === null) return;
        this.actualizePrev(this.prevSibling());
        this.actualizeNext(this.nextSibling());
    }

    actualizePrev(elem = null) {
        if (this.width('px') === null) return;
        if (elem === null) elem = this.prevSibling();
        if (elem === undefined) return;

        if (this.getDirection() == lx.VERTICAL) {
            elem.setGeomPriority(lx.TOP, lx.HEIGHT);
            elem.setGeom([0, 0, undefined, this.top('px') - elem.top('px') + 'px', 0]);
        } else {
            elem.setGeomPriority(lx.LEFT, lx.WIDTH);
            elem.setGeom([0, 0, this.left('px') - elem.left('px') + 'px', undefined, undefined, 0]);
        }
    }

    actualizeNext(elem = null) {
        if (this.width('px') === null) return;
        if (elem === null) elem = this.nextSibling();
        if (elem === undefined) return;

        if (this.getDirection() == lx.VERTICAL) {
            let newH = elem.height('px') + elem.top('px') - (this.top('px') + this.height('px')) + 'px';
            elem.setGeomPriority(lx.TOP, lx.HEIGHT);
            elem.setGeom([0, this.top('px') + this.height('px') + 'px', undefined, newH, 0]);


        } else {
            let newW = elem.width('px') + elem.left('px') - (this.left('px') + this.width('px')) + 'px';
            elem.setGeomPriority(lx.LEFT, lx.WIDTH);
            elem.setGeom([this.left('px') + this.width('px') + 'px', 0, newW, undefined, undefined, 0]);
        }
    }

    getPrevLimit() {
        let match = false,
            prev = this.prevSibling();
        while (prev && !match) {
            if (prev)
                if (lx.isInstance(prev, lx.JointMover)) match = true;
                else prev = prev.prevSibling();
        }

        if (prev)
            return this.getDirection() == lx.VERTICAL
                ? prev.top('px') + prev.height('px') + this.limit
                : prev.left('px') + prev.width('px') + this.limit;

        return this.limit;
    }

    getNextLimit() {
        let match = false,
            next = this.nextSibling();
        while (next && !match) {
            if (next)
                if (lx.isInstance(next, lx.JointMover)) match = true;
                else next = next.nextSibling();
        }

        if (next)
            return this.getDirection() == lx.VERTICAL
                ? next.top('px') - this.limit
                : next.left('px') - this.limit;

        return this.getDirection() == lx.VERTICAL
            ? this.parent.height('px') - this.limit
            : this.parent.width('px') - this.limit;
    }
}
