// @lx:module lx.ColorPicker;
// @lx:module-data: i18n = i18n.yaml;

// @lx:use lx.Color;
// @lx:use lx.Input;
// @lx:use lx.Button;
// @lx:use lx.Scroll;

const __slices = [
    {base: [1, 0, 0], change: 1, direction: +1},
    {base: [1, 1, 0], change: 0, direction: -1},
    {base: [0, 1, 0], change: 2, direction: +1},
    {base: [0, 1, 1], change: 1, direction: -1},
    {base: [0, 0, 1], change: 0, direction: +1},
    {base: [1, 0, 1], change: 2, direction: -1}
];
const __lims = {
    R: [0, 255],
    G: [0, 255],
    B: [0, 255],
    H: [0, 360],
    S: [1, 100],
    L: [1, 100]
};

/**
 * @widget lx.ColorPicker
 * @content-disallowed
 *
 * @events [
 *     colorSelect {newColor: lx.Color, oldColor: lx.Color},
 *     colorChange {newColor: lx.Color, oldColor: lx.Color},
 *     colorReject {oldColor: lx.Color, newColor: lx.Color}
 * ]
 * 
 * CSS classes:
 * - lx-ColorPicker
 */
// @lx:namespace lx;
class ColorPicker extends lx.Box {
    getBasicCss() {
        return 'lx-ColorPicker';
    }

    static initCss(css) {
        css.inheritClass('lx-ColorPicker', 'AbstractBox');
        css.addClass('lx-ColorPicker-pick', {
            cursor: 'pointer'
        });
        css.addClass('lx-ColorPicker-handler', {
            backgroundColor: 'white',
            borderRadius: '50%',
            cursor: 'ew-resize'
        });
        css.addClass('lx-ColorPicker-line', {
            border: '1px white solid'
        });
    }

    render(config={}) {
        super.render(config);
        this.addClass('lx-ColorPicker');
    }

    // @lx:<context CLIENT:
    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [color] {lx.Color}
     * }}
     */
    clientRender(config) {
        const _t = this;
        this._movingLines = false;
        this._movingHue = false;

        this.colorSave = new lx.Color(config.color || '#ff0000');
        this.color = new lx.Color(config.color || '#ff0000');
        this.baseColor = {R: 0, G: 0, B: 0};
        this.colorModel = lx.BindableModel.create({
            R: {default: _t.color.R},
            G: {default: _t.color.G},
            B: {default: _t.color.B},
            RGB: {ref: '_rgb()'},
            H: {default: Math.round(_t.color.getHue())},
            S: {default: Math.round(_t.color.getSaturation())},
            L: {default: Math.round(_t.color.getLightness())},
            text: {default: _t.color.toString()}
        });
        this.colorModel._rgb = function (val) {
            if (val === undefined) return [this.R, this.G, this.B];
            val = lx.Color.castToRgb(val, 255);
            this.R = val[0];
            this.G = val[1];
            this.B = val[2];
        };
        this.colorModel.beforeSet(function(fieldName, value) {
            if (fieldName == 'text' || fieldName == 'RGB') return value;
            if (!lx.isNumber(value) || value < __lims[fieldName][0]) return __lims[fieldName][0];
            if (value > __lims[fieldName][1]) return __lims[fieldName][1];
            return Math.round(value);
        });
        this.colorModel.afterSet(function(fieldName, value) {
            let e = _t.newEvent();
            e.oldColor = _t.color.clone();
            switch (fieldName) {
                case 'R': _t.color.R = this.R; _actualizeColorModel(_t); break;
                case 'G': _t.color.G = this.G; _actualizeColorModel(_t); break;
                case 'B': _t.color.B = this.B; _actualizeColorModel(_t); break;
                case 'RGB':
                    _t.color.R = this.R;
                    _t.color.G = this.G;
                    _t.color.B = this.B;
                    _actualizeColorModel(_t);
                    break;
                case 'text': _t.color.init(this.text); _actualizeColorModel(_t); break;
                case 'H': _t.color.setHue(this.H); _actualizeColorModel(_t); break;
                case 'S': _t.color.setSaturation(this.S); _actualizeColorModel(_t); break;
                case 'L': _t.color.setLightness(this.L); _actualizeColorModel(_t); break;
            }
            e.newColor = _t.color.clone();
            _t.trigger('colorChange', e);
            _onColorChange(_t);
        });

        this.addContainer();
        this.addStructure(lx.Scroll, {type: lx.VERTICAL});
        this.addStructure(lx.Scroll, {type: lx.HORIZONTAL});
        this.gridProportional({indent: '10px', cols: 18});
        this.on('resize', ()=>__actualizeHandlers(this));

        __renderHueScale(this);
        __renderMainScale(this);

        __renderPick(this, 'S', [9, 1, 6, 1]);
        __renderPick(this, 'L', [9, 2, 6, 1]);
        __renderPick(this, 'R', [9, 3, 6, 1]);
        __renderPick(this, 'G', [9, 4, 6, 1]);
        __renderPick(this, 'B', [9, 5, 6, 1]);

        this.add(lx.Input, {field: 'H', geom: [15, 0, 3, 1]});
        this.add(lx.Input, {field: 'S', geom: [15, 1, 3, 1]});
        this.add(lx.Input, {field: 'L', geom: [15, 2, 3, 1]});
        this.add(lx.Input, {field: 'R', geom: [15, 3, 3, 1]});
        this.add(lx.Input, {field: 'G', geom: [15, 4, 3, 1]});
        this.add(lx.Input, {field: 'B', geom: [15, 5, 3, 1]});
        this.add(lx.Input, {field: 'text', geom: [9, 6, 6, 1]});
        this.add(lx.Box, { key: 'colorExample', geom: [15, 6, 3, 1]});

        this.add(lx.Button, {
            geom: [0, 7, 6, 1],
            text: lx.i18n('Select'),
            click: ()=>this.trigger('colorSelect', this.newEvent({
                newColor: this.color.clone(),
                oldColor: this.colorSave.clone()
            }))
        });
        this.add(lx.Button, {
            geom: [6, 7, 6, 1],
            text: lx.i18n('Reset'),
            click: ()=>{
                let oldColor = this.color.clone();
                this.color.copy(this.colorSave);
                this.trigger('colorChange', this.newEvent({
                    oldColor,
                    newColor: this.color.clone()
                }));
                _actualizeColorModel(this);
                _onColorChange(this);
            }
        });
        this.add(lx.Button, {
            geom: [12, 7, 6, 1],
            text: lx.i18n('Reject'),
            click: ()=>this.trigger('colorReject', this.newEvent({
                oldColor: this.colorSave.clone(),
                newColor: this.color.clone()
            }))
        });

        this.bind(this.colorModel);
        _onColorChange(this);
    }

    setColor(color) {
        this.colorSave = new lx.Color(color);
        this.color = new lx.Color(color);
        this.baseColor = {R: 0, G: 0, B: 0};
        this.colorModel.RGB = color;
    }
    // @lx:context>
}

// @lx:<context CLIENT:
function __renderHueScale(self) {
    const hueScale = self.add(lx.Box, {key: 'hueScale', geom: [0, 0, 15, 1], css: 'lx-ColorPicker-pick'});
    const hueGradient = hueScale.add(lx.Box, {key: 'hueGradient', geom: [0, 0, '100%', null, null, '15px']});
    hueGradient.add(lx.Box, {geom: [0, 0, '17%', '100%'], style: {background: 'linear-gradient(to right, #F00, #FF0)'}});
    hueGradient.add(lx.Box, {geom: ['17%', 0, '16%', '100%'], style: {background: 'linear-gradient(to right, #FF0, #0F0)'}});
    hueGradient.add(lx.Box, {geom: ['33%', 0, '17%', '100%'], style: {background: 'linear-gradient(to right, #0F0, #0FF)'}});
    hueGradient.add(lx.Box, {geom: ['50%', 0, '17%', '100%'], style: {background: 'linear-gradient(to right, #0FF, #00F)'}});
    hueGradient.add(lx.Box, {geom: ['67%', 0, '16%', '100%'], style: {background: 'linear-gradient(to right, #00F, #F0F)'}});
    hueGradient.add(lx.Box, {geom: ['83%', 0, '17%', '100%'], style: {background: 'linear-gradient(to right, #F0F, #F00)'}});
    const track = hueScale.add(lx.Box, {key: 'hueTrack', geom: [0, null, '100%', '15px', null, 0]});
    const handler = track.add(lx.Box, {key: 'hueHandler', geom: [0, 0, '15px', '15px'], css: 'lx-ColorPicker-handler'});

    handler.move({yMove: false});
    handler.on('move', ()=>__defineBaseColorByHueScale(self));
    hueScale.click(e=>{
        let g = hueScale.getGlobalRect(),
            pos = Math.round(e.clientX - g.left);
        if (pos > hueScale.width('px') - handler.width('px'))
            pos = hueScale.width('px') - handler.width('px');
        handler.left(pos + 'px');
        __defineBaseColorByHueScale(self);
    });
}

function __renderMainScale(self) {
    const mainScale = self.add(lx.Box, {key: 'mainScale', geom: [0, 1, 9, 6]});
    const vLine = mainScale.add(lx.Rect, {key: 'vLine', geom: [10, 0, '3px', 100], css: 'lx-ColorPicker-line'});
    const hLine = mainScale.add(lx.Rect, {key: 'hLine', geom: [0, 10, 100, '3px'], css: 'lx-ColorPicker-line'});
    mainScale.on('resize', e=>{
        let p = _getScalePoint(self, e.oldWidth, e.oldHeight);
        _actualizeMainScalePoint(self, p.x ,p.y);
    });

    function moveLines(e) {
        let bar = mainScale.getGlobalRect(),
            x = e.clientX - bar.left,
            y = e.clientY - bar.top,
            w = mainScale.width('px') - vLine.width('px'),
            h = mainScale.height('px') - hLine.height('px');
        if (x < 0) x = 0; else if (x > w) x = w;
        if (y < 0) y = 0; else if (y > h) y = h;
        vLine.left(Math.round(x) + 'px');
        hLine.top(Math.round(y) + 'px');
        _scaleColor(self);
    }
    function upHandler(e) {
        self._movingLines = false;
        lx.off('mouseup', upHandler);
    }
    mainScale.on('mousedown', e=>{
        self._movingLines = true;
        moveLines(e);
        lx.on('mouseup', upHandler);
    });
    mainScale.on('mousemove', e=>{
        if (self._movingLines) moveLines(e);
    });
}

function __renderPick(self, field, geom) {
    const pick = self.add(lx.Box, {key: 'pick' + field, geom, css: 'lx-ColorPicker-pick'});
    const gradient = pick.add(lx.Box, {key: 'gradient', geom: [0, 0, '100%', null, null, '15px']});
    switch (field) {
        case 'R':
            gradient.style('background', 'linear-gradient(to right, #FFF, #F00)');
            gradient.setAttribute('title', 'Red');
            break;
        case 'G':
            gradient.style('background', 'linear-gradient(to right, #FFF, #0F0)');
            gradient.setAttribute('title', 'Green');
            break;
        case 'B':
            gradient.style('background', 'linear-gradient(to right, #FFF, #00F)');
            gradient.setAttribute('title', 'Blue');
            break;
        case 'S':
            gradient.setAttribute('title', 'Saturation');
            break;
        case 'L':
            gradient.setAttribute('title', 'Lightness');
            break;
    }
    const grTrack = pick.add(lx.Box, {key: 'track', geom: [0, null, '100%', '15px', null, 0]});
    const grHandler = grTrack.add(lx.Box, {key: 'handler', geom: [0, 0, '15px', '15px'], css: 'lx-ColorPicker-handler'});
    grHandler.move({yMove: false});
    grHandler.on('move', ()=>{
        let pos = grHandler.left('px') / (grTrack.width('px') - grHandler.width('px'));
        self.colorModel[field] = pos * __lims[field][1];
    });
    pick.click(function (e) {
        if (e.target === grHandler.getDomElem()) return;
        let g = pick.getGlobalRect(),
            pos = Math.round(e.clientX - g.left);
        self.colorModel[field] = (pos / pick.width('px')) * __lims[field][1];
    });
}

function __defineBaseColorByHueScale(self) {
    const hueScale = lx(self)>>hueScale;
    const hueHandler = lx(self)>>hueHandler;

    const map = [17, 16, 17, 17, 16, 17];
    let pos = (hueHandler.left('px') / (hueScale.width('px') - hueHandler.width('px'))) * 100,
        shift = 0,
        sliceIndex, sliceShift;
    for (let i=0; i<6; i++) {
        let delta = map[i];
        if (pos >= shift && pos <= shift + delta) {
            sliceIndex = i;
            sliceShift = pos - shift;
            break;
        }
        shift += delta;
    }
    sliceShift = sliceShift / map[sliceIndex];

    let slice = __slices[sliceIndex],
        rgb = [slice.base[0], slice.base[1], slice.base[2]];
    rgb[slice.change] += sliceShift * slice.direction;

    self.baseColor.R = Math.round(rgb[0] * 255);
    self.baseColor.G = Math.round(rgb[1] * 255);
    self.baseColor.B = Math.round(rgb[2] * 255);
    self._movingHue = true;
    _actualizeMainScale(self);
    _scaleColor(self);
    self._movingHue = false;
}

function _defineSliceIndex(self) {
    let sliceIndex = 0;
    for (let i=0; i<6; i++) {
        let slice = __slices[i],
            base = slice.base,
            sliceMask = [base[0], base[1], base[2]],
            colorMask = [2, 2, 2];
        sliceMask[slice.change] = 2;
        if (self.baseColor.R === 255) colorMask[0] = 1;
        else if (self.baseColor.G === 255) colorMask[1] = 1;
        else if (self.baseColor.B === 255) colorMask[2] = 1;
        if (self.baseColor.R === 0) colorMask[0] = 0;
        else if (self.baseColor.G === 0) colorMask[1] = 0;
        else if (self.baseColor.B === 0) colorMask[2] = 0;
        if (colorMask[0] == sliceMask[0] && colorMask[1] == sliceMask[1] && colorMask[2] == sliceMask[2]) {
            sliceIndex = i;
            break;
        }
    }
    return sliceIndex;
}

function __defineBaseColor(self) {
    // Calculate base color and main scale coordinates
    function calc(c1, c2, c3) {
        let y = (255 - c1) / 255;
        if (y == 1) return {x: -1, y: -1, color: -1};
        let x = c2 / (255 * (1 - y));
        let divider = x - 1 - x * y + y;
        if (divider == 0) return {x: -1, y: -1, color: -1};
        let c = (255 * x - 255 * x * y - c3) / divider;
        return { x, y, color: Math.round(c) };
    }
    let color = {R: 0, G: 0, B: 0},
        x, y,
        sequances = [
            ['R', 'G', 'B'],
            ['R', 'B', 'G'],
            ['G', 'B', 'R'],
            ['G', 'R', 'B'],
            ['B', 'R', 'G'],
            ['B', 'G', 'R']
        ];
    for (let i=0; i<6; i++) {
        let sequance = sequances[i],
            c0 = sequance[0],
            c1 = sequance[1],
            c2 = sequance[2],
            res = calc(self.colorModel[c0], self.colorModel[c1], self.colorModel[c2]);
        if (res.x >= 0 && res.x <= 1 && res.y >= 0 && res.y <= 1 && res.color >= 0 && res.color <= 255) {
            x = res.x;
            y = res.y;
            color[c0] = 255;
            color[c1] = 0;
            color[c2] = res.color;
            break;
        }
    }
    if (color.R !== 0 || color.G !== 0 || color.B !== 0) {
        self.baseColor.R = color.R;
        self.baseColor.G = color.G;
        self.baseColor.B = color.B;
    }

    _actualizeHueScale(self);
    _actualizeMainScale(self);
    _actualizeMainScalePoint(self, x, y);
}

function _onColorChange(self) {
    _actualizeGrHandler(lx(self)>>pickR, 'R', self.colorModel.R);
    _actualizeGrHandler(lx(self)>>pickG, 'G', self.colorModel.G);
    _actualizeGrHandler(lx(self)>>pickB, 'B', self.colorModel.B);
    _actualizeGrHandler(lx(self)>>pickS, 'S', self.colorModel.S);
    _actualizeGrHandler(lx(self)>>pickL, 'L', self.colorModel.L);
    _actualizeSaturation(self);
    _actualizeLightness(self);
    if (!self._movingLines && !self._movingHue)
        __defineBaseColor(self);

    lx(self)>>colorExample.fill(self.color);
}

function __actualizeHandlers(self) {
    _actualizeHueScale(self);
    _actualizeGrHandler(lx(self)>>pickR, 'R', self.colorModel.R);
    _actualizeGrHandler(lx(self)>>pickG, 'G', self.colorModel.G);
    _actualizeGrHandler(lx(self)>>pickB, 'B', self.colorModel.B);
    _actualizeGrHandler(lx(self)>>pickS, 'S', self.colorModel.S);
    _actualizeGrHandler(lx(self)>>pickL, 'L', self.colorModel.L);
}

function _actualizeColorModel(self) {
    const colorModel = self.colorModel;
    colorModel.ignoreSetterListener(true);
    colorModel.R = self.color.R;
    colorModel.G = self.color.G;
    colorModel.B = self.color.B;
    colorModel.H = Math.round(self.color.getHue());
    colorModel.S = Math.round(self.color.getSaturation());
    colorModel.L = Math.round(self.color.getLightness());
    colorModel.text = self.color.toString();
    colorModel.ignoreSetterListener(false);
    colorModel.bindRefresh();
}

function _actualizeSaturation(self) {
    const gradient = lx(self)>>pickS>gradient;
    let color0 = self.color.clone(),
        color1 = self.color.clone();
    color0.setSaturation(1);
    color1.setSaturation(100);
    gradient.style('background', 'linear-gradient(to right, '+color0+', '+color1+')');
}

function _actualizeLightness(self) {
    const gradient = lx(self)>>pickL>gradient;
    let color0 = self.color.clone(),
        color1 = self.color.clone();
    color0.setLightness(1);
    color1.setLightness(100);
    gradient.style('background', 'linear-gradient(to right, '+color0+', '+color1+')');
}

function _actualizeGrHandler(gr, field, value) {
    const grTrack = lx(gr)>>track;
    const grHandler = lx(gr)>>handler;
    let pos = Math.round(value * (grTrack.width('px') - grHandler.width('px')) / __lims[field][1]);
    if (pos > grTrack.width('px') - grHandler.width('px'))
        pos = grTrack.width('px') - grHandler.width('px');
    grHandler.left(pos + 'px');
}

function _actualizeMainScale(self) {
    const mainScale = lx(self)>>mainScale;
    mainScale.style(
        'background',
        'linear-gradient(to bottom, rgba(255,255,255,0), rgba(0,0,0,1))'
        + ','
        + 'linear-gradient(to right, rgba(255,255,255), rgba('
        +self.baseColor.R+','+self.baseColor.G+','+self.baseColor.B+ '))'
    )
}

function _actualizeMainScalePoint(self, x ,y) {
    if (x === undefined)
        if (self.colorModel.R > 250) x = 1;
        else x = 0;
    if (y === undefined)
        if (self.colorModel.R > 250) y = 0;
        else y = 1;
    x = 1 - x;
    const mainScale = lx(self)>>mainScale;
    const vLine = lx(mainScale)>>vLine;
    const hLine = lx(mainScale)>>hLine;
    vLine.left(Math.round((mainScale.width('px') - vLine.width('px')) * x) + 'px');
    hLine.top(Math.round((mainScale.height('px') - hLine.height('px')) * y) + 'px');
}

function _actualizeHueScale(self) {
    let sliceIndex = _defineSliceIndex(self),
        partColor = self.baseColor[['R', 'G', 'B'][__slices[sliceIndex].change]];
    if (__slices[sliceIndex].direction == -1) partColor = 255 - partColor;
    const hueGradient = lx(self)>>hueGradient;
    const hueHandler = lx(self)>>hueHandler;
    const sliceBox = hueGradient.child(sliceIndex);
    let left = Math.round((sliceBox.width('px') - hueHandler.width('px')) * partColor / 255
        + sliceBox.left('px'));
    hueHandler.left(left + 'px');
}

function _scaleColor(self) {
    let p = _getScalePoint(self);
    self.colorModel.R = ((255 - self.baseColor.R) * p.x + self.baseColor.R) * (1 - p.y);
    self.colorModel.G = ((255 - self.baseColor.G) * p.x + self.baseColor.G) * (1 - p.y);
    self.colorModel.B = ((255 - self.baseColor.B) * p.x + self.baseColor.B) * (1 - p.y);
}

function _getScalePoint(self, w = null, h = null) {
    const mainScale = lx(self)>>mainScale;
    const vLine = lx(mainScale)>>vLine;
    const hLine = lx(mainScale)>>hLine;
    let x = vLine.left('px'),
        y = hLine.top('px');
    if (w === null) w = mainScale.width('px');
    if (h === null) h = mainScale.height('px');

    w -= vLine.width('px');
    h -= hLine.height('px');

    x = (w - x) / w;
    y = y / h;
    if (x > 1) x = 1;
    else if (x < 0) x = 0;
    if (y > 1) y = 1;
    else if (y < 0) y = 0;
    return {x, y};
}
// @lx:context>
