let _x, _y,
    _down = new Map(),
    _move = new Map(),
    _up = new Map();

// @lx:namespace lx;
class Mouse extends lx.AppComponent {
    get x() { return _x; }
    get y() { return _y; }

    getPosition(context = null) {
        if (!context) {
            return {
                x: this.x,
                y: this.y
            };
        }

        let rect = context.getGlobalRect();
        return {
            x: this.x - rect.left,
            y: this.y - rect.top
        };
    }

    onDown(f) {
        _down.set(f, f);
    }

    onMove(f) {
        _move.set(f, f);
    }

    onUp(f) {
        _up.set(f, f);
    }

    offDown(f) {
        _down.delete(f);
    }

    offMove(f) {
        _move.delete(f);
    }

    offUp(f) {
        _up.delete(f);
    }
    
    onReady() {
        document.body.addEventListener('mousemove', e=>{
            _x = e.clientX;
            _y = e.clientY;
            _move.forEach(f=>f(e));
        });
        document.body.addEventListener('touchmove', e=>{
            _x = e.clientX;
            _y = e.clientY;
            _move.forEach(f=>f(e));
        });

        document.addEventListener('mousedown', e=>_down.forEach(f=>f(e)));
        document.addEventListener('touchstart', e=>_down.forEach(f=>f(e)));
        document.addEventListener('mouseup', e=>_up.forEach(f=>f(e)));
        document.addEventListener('touchend', e=>_up.forEach(f=>f(e)));
    }
}
