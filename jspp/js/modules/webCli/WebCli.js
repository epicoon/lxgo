// @lx:module lx.WebCli;
// @lx:module-data: backend = lx_WebCli;

// @lx:use lx.Plugin;
// @lx:use lx.ActiveBox;
// @lx:require -R classes/;

// @lx:namespace lx;
class WebCli extends lx.Element {
    constructor(box) {
        super();

        Console.init(box);
        box.addClass('lxWC-back');

        this.commandList = [];

        this.service = null;
        this.plugin = null;

        this.inputString = '';
        this.commandsHistory = [];
        this.commandsHistoryIndex = 0;

        this.processParams = {};
        this.inProcess = false;
        this.processNeed = null;

        Console.useCache = true;
        Console.input(this.getLocationText());
        Console.setCallback('Enter', [this, this.onEnter]);
        Console.setCallback('Tab', [this, this.onTab]);
        Console.setCallback('ArrowUp', [this, this.onUp]);
        Console.setCallback('ArrowDown', [this, this.onDown]);

        lx(server).getCommandList().then(res=>this.commandList=res.data);
    }

    static initCss(css) {
        css.addClass('lxWC-back', {
            padding: '10px',
            fontFamily: 'Courier', // Verdana
            tabSize: 4,
            backgroundColor: '#272822',
            color: '#F8F8F2'
        });
        css.addClass('.lxWC-loc', {
            fontWeight: 'bold',
            color: '#FFCF00'
        });
        css.addClass('.lxWC-command', {
            color: '#FF9900'
        });
        css.addClass('.lxWC-msg_', {
        });
        css.addClass('.lxWC-msg_b', {
            fontWeight: 'bold'
        });
        css.addClass('.lxWC-msg_u', {
            textDecoration: 'underline'
        });
        css.addClass('.lxWC-msg_bu', {
            fontWeight: 'bold',
            textDecoration: 'underline'
        });
        css.addClass('.lxWC-selected', {
            fontWeight: 'bold',
            textDecoration: 'underline'
        });
    }

    getLocationText() {
        var text;
        if (this.plugin) {
            text = 'lx-cli&lt;plugin:' + this.plugin + '&gt;:';
        } else if (this.service) {
            text = 'lx-cli&lt;service:' + this.service + '&gt;:';
        } else {
            text = 'lx-cli&lt;app&gt;:';
        }
        return text;
    }

    onEnter() {
        if (this.inProcess) {
            this.processParams[this.processNeed] = Console.command;
            this.handleCommand(this.inProcess);
            return;
        }

        var input = Console.command;
        if (input === undefined || input == '') {
            Console.input(this.getLocationText());
            return;
        }

        if (!this.commandsHistory.len || input != this.commandsHistory.lxLast()) {
            this.commandsHistory.push(input);
            this.commandsHistoryIndex = this.commandsHistory.len;
        }

        var command = this.identifyCommandKey(input);
        this.inputString = input;

        if (command == 'clear') {
            this.clearConsole();
            Console.input(this.getLocationText());
            return;
        }

        if (!this.validateCommand(command)) {
            Console.outln("Unknown command '" + command +"'. Enter 'help' to see the commands list");
            Console.outCache();
            Console.input(this.getLocationText());
            return;
        }

        this.handleCommand(command);
    }

    onTab() {
        var currentInput = Console.getCurrentInput();
        lx(server).tryFinishCommand(currentInput).then(result=>{
            if (!result.success) return;
            var command = result.data;
            if (!command) return;
            if (command.common == currentInput) {
                Console.outln();
                Console.outln(command.matches.join('  '));
                Console.outCache();
                Console.input(this.getLocationText());
                Console.replaceInput(currentInput);
            } else {
                Console.replaceInput(command.common);
            }
        });
    }

    onUp() {
        if (this.commandsHistoryIndex == 0) return;
        this.commandsHistoryIndex--;
        Console.replaceInput(this.commandsHistory[this.commandsHistoryIndex]);
    }

    onDown() {
        if (this.commandsHistoryIndex == this.commandsHistory.len) {
            return;
        }
        this.commandsHistoryIndex++;
        if (this.commandsHistoryIndex == this.commandsHistory.len) {
            Console.replaceInput('');
            return;
        }
        Console.replaceInput(this.commandsHistory[this.commandsHistoryIndex]);
    }

    /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
     * HANDLERS
     * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

    handleCommand(command) {
        lx(server).handleCommand(command, this.inputString, this.processParams, this.service, this.plugin).then(result=>{
            if (!result.success) {
                Console.outln(result.data);
                Console.outCache();
                Console.input(this.getLocationText());
                return;
            }

            result = result.data;

            if (result.data && result.data.code == 'ext') {
                this.handleExtendedCommand(result.data);
                return;
            }

            for (var key in result.params) {
                var value = result.params[key];
                this.processParams[key] = value;
            }
            for (var i in result.invalidParams) {
                var name = result.invalidParams[i];
                delete this.processParams[name];
            }
            for (var i in result.output) {
                var row = result.output[i];
                var decor = '';
                if (row[2] && row[2].decor) decor = row[2].decor;
                if (row[0] == 'in') {
                    Console.outCache();
                    this.processNeed = result.need;
                    this.inProcess = command;
                    Console.input(row[1], decor);
                    return;
                } else if (row[0] == 'select') {
                    Console.outCache();
                    this.processNeed = result.need;
                    this.inProcess = command;
                    Console.select({
                        hintText: row[2],
                        hintDecor: row[3],
                        options: row[1]
                    });
                    return;
                }
                Console[row[0]](row[1], decor);
            }
            Console.outCache();

            if (result.keepProcess) {
                this.inProcess = command;
            } else {
                this.inProcess = false;
                this.processNeed = null;
                this.processParams = {};
                this.service = result.service;
                this.plugin = result.plugin;
                Console.input(this.getLocationText());
            }
        });
    }

    handleExtendedCommand(data) {
        if (data.message) {
            Console.outln(data.message);
            Console.outCache();
        }
        Console.input(this.getLocationText());
        Console.checkCarret();

        if (data.type == 'plugin') {
            var ab = new lx.ActiveBox({
                parent: lx.app.root,
                geom: true,
                header: data.header,
                closeButton: {click:()=>ab.del()}
            });
            ab.get('body').setPlugin(data.plugin);
        }
    }

    clearConsole() {
        Console.clear();
    }

    /* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
     * BASIC
     * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

    identifyCommandKey(input) {
        var arr = input.split(' ');
        var command = arr.shift();
        return command;
    }

    validateCommand(commandName) {
        for (var i=0, l=this.commandList.len; i<l; i++)
            if (this.commandList[i].command.includes(commandName))
                return true;
        return false;
    }
}
