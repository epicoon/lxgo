// @lx:module lx.MdHighlighter;

// @lx:namespace lx;
class MdHighlighter extends lx.Element {
    static initCss(css) {
        css.addClass('lx-md-container', {
            padding: '20px'
        });
        css.addClass('lx-md-paragraph', {
            margin: 0,
            paddingTop: '10px',
            paddingBottom: '10px'
        });
        css.addClass('lx-md-codeblock', {
            marginTop: '20px',
            marginBottom: '20px',
            padding: '20px',
            backgroundColor: css.preset.bodyBackgroundColor,
            color: css.preset.textColor
        });
        css.addClass('lx-md-blockquote', {
            margin: 0,
            marginTop: '20px',
            marginBottom: '20px',
            paddingLeft: '20px',
            backgroundColor: css.preset.bodyBackgroundColor,
            color: css.preset.textColor,
            borderLeft: 'solid 2px' + css.preset.coldSoftColor
        });
        css.addClass('lx-md-table', {
            marginTop: '20px',
            marginBottom: '20px',
            border: 'solid 1px ' + css.preset.widgetBorderColor,
            borderCollapse: 'collapse'
        });
        css.addClass('lx-md-table-header', {
            padding: '10px',
            border: 'solid 1px ' + css.preset.widgetBorderColor,
        });
        css.inheritClass('lx-md-table-cell', 'lx-md-table-header');

        css.addClass('lx-md-token-keyword', {
            color: css.preset.coldMainColor
        });
        css.addClass('lx-md-token-string', {
            color: css.preset.checkedMainColor
        });
        css.addClass('lx-md-token-comment', {
            color: css.preset.textColorDisabled
        });
        css.addClass('lx-md-token-number', {
            color: css.preset.hotMainColor
        });
    }

    // Keyword lists per code-type. Add a language by adding an entry here -
    // the tokenizer itself doesn't need to change.
    static _keywords() {
        return {
            js: [
                'const', 'let', 'var', 'function', 'return', 'if', 'else', 'for', 'while', 'do',
                'switch', 'case', 'break', 'continue', 'class', 'extends', 'new', 'this', 'typeof',
                'instanceof', 'in', 'of', 'try', 'catch', 'finally', 'throw', 'async', 'await',
                'yield', 'static', 'get', 'set', 'import', 'export', 'default', 'from', 'null',
                'undefined', 'true', 'false', 'void', 'delete',
            ],
            go: [
                'package', 'import', 'func', 'return', 'if', 'else', 'for', 'range', 'switch',
                'case', 'break', 'continue', 'var', 'const', 'type', 'struct', 'interface', 'map',
                'chan', 'go', 'defer', 'select', 'fallthrough', 'goto', 'nil', 'true', 'false',
            ],
        };
    }

    // A single combined regex per code-type, with one named group per token
    // category. Comment and string come before number/keyword so that text
    // inside them is never re-tokenized. Line comments stop at "<br>" or the
    // end of the string - the code text has no real newlines, lines are
    // joined with a literal "<br>" tag, so an unbounded ".*" would swallow
    // every following line instead of just the current one.
    static _buildRule(keywords) {
        const kw = keywords.join('|');
        const pattern = [
            '(?<comment>\\/\\*[\\s\\S]*?\\*\\/|\\/\\/.*?(?=<br>|$))',
            '(?<string>"(?:[^"\\\\]|\\\\.)*"|\'(?:[^\'\\\\]|\\\\.)*\'|`(?:[^`\\\\]|\\\\.)*`)',
            '(?<number>\\b\\d+(?:\\.\\d+)?\\b)',
            `(?<keyword>\\b(?:${kw})\\b)`,
        ].join('|');
        return new RegExp(pattern, 'g');
    }

    static _rules() {
        if (!MdHighlighter._rulesCache) {
            const keywords = MdHighlighter._keywords();
            const rules = {};
            for (const codeType in keywords) {
                rules[codeType] = MdHighlighter._buildRule(keywords[codeType]);
            }
            MdHighlighter._rulesCache = rules;
        }
        return MdHighlighter._rulesCache;
    }

    static highlight(tag) {
        const codeType = tag.getAttribute('code-type');
        const rule = MdHighlighter._rules()[codeType];
        if (!rule) return;

        tag.innerHTML = tag.innerHTML.replace(rule, (match, ...rest) => {
            const groups = rest[rest.length - 1];
            let category = null;
            if (groups.comment !== undefined) category = 'comment';
            else if (groups.string !== undefined) category = 'string';
            else if (groups.number !== undefined) category = 'number';
            else if (groups.keyword !== undefined) category = 'keyword';

            if (!category) return match;
            return `<span class="lx-md-token-${category}">${match}</span>`;
        });
    }
}
