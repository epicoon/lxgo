let _x, _y;

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
    
    onReady() {
        document.body.addEventListener('mousemove', (e) => {
            _x = e.clientX;
            _y = e.clientY;
        });
    }
}
