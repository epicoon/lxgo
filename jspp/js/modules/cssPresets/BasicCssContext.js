// @lx:module lx.BasicCssContext;

// @lx:namespace lx;
class BasicCssContext extends lx.CssContextHolder {
    static init(css) {
        css.registerMixin('img', url => {
            return {
                backgroundImage: 'url(' + url + ')',
                backgroundRepeat: 'no-repeat',
                backgroundSize: '100% 100%'
            };
        });

        css.registerMixin('ellipsis', ()=>{
            return {
                overflow: 'hidden',
                whiteSpace: 'nowrap',
                textOverflow: 'ellipsis'
            }
        });

        /**
         * @param iconCode {String}
         * @param [config] {Number|Object: {
         *     [fontSize = 'calc(30px + 1.0vh)'] {Number|String},
         *     [fontWeight = 500] {Number|String},
         *     [color = 'inherit'] {String}
         *     [fontFamily = 'inherit'] {String}
         * }}
         * Examples:
         * '@icon': {'\\2297', 16}
         * '@icon': {'\\21BB', {fontWeight: 300, fontSize: 16}}
         */
        css.registerMixin('icon', (iconCode, config = null) => {
            var iconFlex = {
                display: 'flex',
                flexDirection: 'row',
                alignItems: 'center',
                justifyContent: 'center'
            };
            var iconStyle = {
                fontSize: 'calc(30px + 1.0vh)',
                fontWeight: '500',
                color: 'inherit',
                fontFamily: 'inherit',
                content: "'" + iconCode + "'"
            };
            if (config) {
                if (lx.isNumber(config)) iconStyle.fontSize = config;
                else if (lx.isObject(config)) iconStyle = iconStyle.lxMerge(config, true);
                if (lx.isNumber(iconStyle.fontSize))
                    iconStyle.fontSize = 'calc(' + iconStyle.fontSize + 'px + 1.0vh)';
            }
            return {
                content: iconFlex,
                pseudoclasses: {
                    after: iconStyle
                }
            };
        });

        css.registerMixin('clickable', () => {
            return {
                content: {
                    marginTop: '0px',
                    cursor: 'pointer',
                    boxShadow: this.presetValue(
                        ['shadowSize', 'shadowSize'],
                        [10, 10],
                        (v1, v2) => {
                            v1 = (v1 * 0.33) + 3;
                            v2 = v1 * 0.5;
                            return '0 ' + v2.toFixed(2) + 'px ' + v1.toFixed(2) + 'px rgba(0,0,0,0.5)';
                        }
                    )
                },
                pseudoclasses: {
                    'hover:not([disabled])': {
                        marginTop: '-2px',
                        boxShadow: this.presetValue(
                            ['shadowSize', 'shadowSize'],
                            [10, 10],
                            (v1, v2) => {
                                v1 = ((v1 * 0.33) + 3) * 1.5;
                                v2 = v1 * 0.5 * 1.5;
                                return '0 ' + v2.toFixed(2) + 'px ' + v1.toFixed(2) + 'px rgba(0,0,0,0.5)';
                            }
                        ),
                        transition: 'margin-top 0.1s linear, box-shadow 0.1s linear',
                    },
                    'active:not([disabled])': {
                        marginTop: '0px',
                        boxShadow: this.presetValue(
                            ['shadowSize', 'shadowSize'],
                            [10, 10],
                            (v1, v2) => {
                                v1 = (v1 * 0.33) + 3;
                                v2 = v1 * 0.5;
                                return '0 ' + v2.toFixed(2) + 'px ' + v1.toFixed(2) + 'px rgba(0,0,0,0.5)';
                            }
                        ),
                        transition: 'margin-top 0.08s linear, box-shadow 0.05s linear'
                    },
                }
            };
        });

        css.addAbstractClass('AbstractBox', {
            borderRadius: this.presetValue('borderRadius', '5px'),
            boxShadow: this.presetValue('shadowSize', 10, val => '0 0px ' + val + 'px rgba(0,0,0,0.5)'),
            backgroundColor: this.presetValue('bodyBackgroundColor', '#3C3F41')
        });

        css.addAbstractClass('Button', {
            overflow: 'hidden',
            whiteSpace: 'nowrap',
            textOverflow: 'ellipsis',
            borderRadius: this.presetValue('borderRadius', '5px'),
            boxShadow: this.presetValue(
                ['shadowSize', 'shadowSize'],
                [10, 10],
                (v1, v2) => {
                    v1 = (v1 * 0.33) + 3;
                    v2 = v1 * 0.5;
                    return '0 ' + v2.toFixed(2) + 'px ' + v1.toFixed(2) + 'px rgba(0,0,0,0.5)';
                }
            ),
            cursor: 'pointer',
            color: this.presetValue('textColor', '#BABABA'),
            backgroundColor: this.presetValue('widgetBackgroundColor', '#5B5E65'),
        });

        css.addAbstractClass('Input', {
            border: this.presetValue('widgetBorderColor', '#646464', val => '1px solid ' + val),
            padding: '4px 5px',
            background: this.presetValue('textBackgroundColor', '#45494A'),
            borderRadius: this.presetValue('borderRadius', '5px'),
            outline: 'none',
            boxShadow: 'inset 0 1px 2px rgba(0, 0, 0, 0.3)',
            fontFamily: 'inherit',
            fontSize: 'calc(10px + 1.0vh)',
            color: this.presetValue('textColor', '#BABABA')
        });

        css.inheritAbstractClass('ActiveButton', 'Button', {
            '@clickable': true
        }, {
            disabled: {
                opacity: '0.5',
                cursor: 'default'
            }
        });
    }
}
