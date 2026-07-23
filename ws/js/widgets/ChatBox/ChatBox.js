// @lx:module lx.socket.ChatBox;
// @lx:module-data: i18n = i18n.yaml;

lx.import(
    lx.BasicCssContext,
    lx.MultiBox,
    lx.JointMover,
    lx.Dropbox,
    lx.Textarea,
    lx.Button,
    lx.Scroll
);

/**
 * @widget lx.MultiBox
 * @content-disallowed
 *
 * @events [
 *     newUnreadMessage,
 *     messageRead
 * ]
 */
// @lx:namespace lx.socket;
class ChatBox extends lx.Box {
    static initCss(css) {
		css.useExtender(lx.BasicCssContext);

        css.inheritClass('lxSocket-ChatBox', 'AbstractBox');
        css.addClass('lxSocket-ChatBox-gear', {
            color: css.preset.widgetIconColor,
            '@icon': ['\\2699', {fontSize:14}],
            cursor: 'pointer'
        });

        css.addAbstractClass('chatMsg', {
            paddingTop: '5px',
            paddingBottom: '5px',
            width: 'fit-content',
            maxWidth: '80%',
            height: '100%',
            borderRadius: css.preset.borderRadius,
            backgroundColor: css.preset.widgetBackgroundColor,
            color: css.preset.widgetIconColor
        });
        css.inheritClass('lxSocket-ChatBox-localMsg', 'chatMsg', {
            float: 'right',
            marginRight: '5%'
        });
        css.inheritClass('lxSocket-ChatBox-outerMsg', 'chatMsg', {
            float: 'left',
            marginLeft: '2%'
        });

        css.addClass('lxSocket-ChatBox-msgTitle', {
            fontSize: '0.7em',
            fontWeight: 'bold'
        });

        css.addClass('lxSocket-ChatBox-indicator', {
            '@icon': ['\\25C9', {fontSize:14}],
            cursor: 'pointer'
        });
        css.addClass('lxSocket-ChatBox-indicator-off', {
            color: css.preset.hotLightColor
        });
        css.addClass('lxSocket-ChatBox-indicator-on', {
            color: css.preset.checkedLightColor
        });

        css.addClass('lxSocket-ChatBox-msg-marker', {
            height: '100%',
            paddingRight: '5px'
        });
        css.addClass('lxSocket-ChatBox-msg-received', {
            '@icon': ['\\2713', {fontSize:12}],
        });
        css.addClass('lxSocket-ChatBox-msg-read', {
            '@icon': ['\\2713', {fontSize:12}],
            color: css.preset.checkedLightColor
        });
    }

    /**
     * @widget-init
     *
     * @param [config] {Object: {
     *     #merge(lx.Rect::constructor::config),
     *     [chatId = 1] {Number|String},
     *     [mateNameField = 'name'] {String},
     *     [matesPosition = lx.TOP] {Number&Enum(
     *         lx.TOP,
     *         lx.BOTTOM,
     *         lx.LEFT,
     *         lx.RIGHT
     *     )},
     * }}
     */
    render(config) {
        super.render(config);

        this.addClass('lxSocket-ChatBox');
        this.chatId = config.chatId || 1;
        this.mateNameField = config.mateNameField || 'name';

        this.streamProportional({direction: lx.VERTICAL});

        const header = this.add(lx.Box, {
            key: 'header',
            height: '50px'
        });

        let wrapper = this.add(lx.Box);
        const body = wrapper.add(lx.MultiBox, {
            key: 'chat',
            geom: [0, 0, null, null, 0, '50px'],
            marks: [ lx.i18n(allLabel) ],
            animation: true,
            joint: true,
            marksStyle: lx.MultiBox.STYLE_STREAM,
            marksPosition: lx.getFirstDefined(config.matesPosition, lx.TOP),
            appendAllowed: true,
            dropAllowed: true
        });
        wrapper.add(lx.JointMover, { bottom: '50px' });
        const footer = wrapper.add(lx.Box, { key: 'footer', geom: true });

        header.gridProportional({indent: '10px', paddingBottom: 0});
        //TODO settings - change name, change marks location
        header.add(lx.Box, {key:'settings', css: 'lxSocket-ChatBox-gear'});
        lx(header)>settings.align(lx.CENTER, lx.MIDDLE);
        header.add(lx.Dropbox, {
            key: 'mateChoice',
            width: 10
        });
        header.add(lx.Box, {key: 'indicator', css: 'lxSocket-ChatBox-indicator'});
        lx(header)>indicator.align(lx.CENTER, lx.MIDDLE);

        //TODO emojes

        footer.gridProportional({indent: '10px', paddingTop: 0, minHeight: '20px'});
        footer.add(lx.Textarea, {
            key: 'message',
            width: 9
        });
        footer.add(lx.Button, {
            key: 'send',
            text: lx.i18n(send),
            width: 3
        });

        _initSheet(this, lx(this)>>chat.sheet(0));
        lx(this)>>chat.mark(0).removeDelButton();
    }

    // @lx:<context CLIENT:
    clientRender(config) {
        super.clientRender(config);

        this.socket = null;
        this.channel = '';
        this.chatList = new lxChatList(this);
        this.indicator = lx.BindableModel.create({
            status: {default: 'disconnected'}
        });
        const indicator = lx(this)>>indicator;
        const self = this;
        indicator.setField('status', function (val) {
            this.removeClass('lxSocket-ChatBox-indicator-off');
            this.removeClass('lxSocket-ChatBox-indicator-on');
            switch (val) {
                case 'disconnected':
                    this.addClass('lxSocket-ChatBox-indicator-off');
                    break;
                case 'connected':
                    this.addClass('lxSocket-ChatBox-indicator-on');
                    break;
            }
        });
        indicator.bind(this.indicator);

        lx(this)>>message.on('keydown', (e)=>{
            if (e.key == 'Enter') {
                if (lx.app.keyboard.shiftPressed()) return;
                e.preventDefault();
                this.sendMessage();
            }
        });
        lx(this)>>send.click(()=>this.sendMessage());
        lx(this)>>mateChoice.on('change', e=>_onChooseMate(self, e.newValue));
        lx(this)>>chat.on('selected', e=>this.chatList.setActive(e.mark));
    }

    setSocket(socket, channel) {
        this.socket = socket;
        this.channel = channel;
        this.socket.onPromisedConnection(()=>_initConnection(this));
    }

    getChannel() {
        if (!this.socket || !this.channel)
            throw 'Channel not found!';
        const ch = this.socket.getChannel(this.channel);
        if (!ch)
            throw 'Channel not found!';
        return ch;
    }

    receiveMessage(message, messageId, senderId, isPrivate) {
        let messageObj = this.chatList.addMessage(message, messageId, senderId, isPrivate);
        if (messageObj.isVisible())
            _sendMessageRead(this, messageObj);
        else
            _sendMessageReceived(this, messageObj);
    }

    sendMessage() {
        const messageBox = lx(this)>>message;
        let message = messageBox.value();
        if (message == '') return;

        if (!this.socket || !this.socket.isConnected()) {
            lx.toastWarning(lx.i18n(noConnection));
            return;
        }
        const channel = this.getChannel();
        if (!channel) {
            lx.toastWarning(lx.i18n(noChannel));
            return;
        }
        
        messageBox.value('');
        let messageObj = this.chatList.addMessage(message);

        let isPrivate = !this.chatList.isCommonActive(),
            receivers = isPrivate ? [this.chatList.active] : null;
        channel.send({
            lxChatBox: this.chatId,
            type: 'message',
            isPrivate,
            message,
            messageId: messageObj.id
        }, receivers);
    }
    // @lx:context>
}


/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

function _initSheet(self, sheet) {
    sheet.addContainer();
    sheet.addStructure(lx.Scroll, {key: 'scroll', type: lx.VERTICAL});
    const stream = sheet.add(lx.Box, {key: 'stream'});
    stream.stream({indent: '10px', height: 'auto'});
}

// @lx:<context CLIENT:
function _prepareMessage(self, message, sender = null) {
    message = message.replace(/(\r|\n|\r\n)/g, '<br>');
    return sender
        ? '<span class="lxSocket-ChatBox-msgTitle">' + sender + ':</span><br>' + message
        : message;
}

function _addMessage(self, chatBox, message, local) {
    let msgRow = lx(chatBox)>>stream.add(lx.Box);
    let msgWrapper = msgRow.add(lx.Box, {
        css: local ? 'lxSocket-ChatBox-localMsg' : 'lxSocket-ChatBox-outerMsg',
    });
    let text = msgWrapper.add(lx.Box, {
        height: '100%',
        text: message
    });
    text.align(lx.LEFT, lx.MIDDLE);
    text.style('float', 'left');

    if (local) {
        text.style('max-width', '90%');
        let marker = msgWrapper.add(lx.Box, {
            css: 'lxSocket-ChatBox-msg-marker'
        });
        marker.style('float', 'right');
        marker.add(lx.Box, {
            key: 'marker',
            size: ['auto', 'auto']
        });
        marker.align(lx.CENTER, lx.BOTTOM);
    }

    if (chatBox.isDisplay()) chatBox.scrollTo({yShift: 1});
    return msgRow;
}

function _initConnection(self) {
    const socket = self.socket;
    self.indicator.status = 'connected';
    _updateMateChoice(self);

    // Handlers
    socket.onChannelMessage(self.channel, message=>{
        const data = message.getData();
        if (!data.lxChatBox || data.lxChatBox != self.chatId) return;
        const sender = message.getAuthor();
        if (sender.isLocal()) return;
        switch (data.type) {
            case 'message':
                self.receiveMessage(data.message, data.messageId, sender.getId(), data.isPrivate);
                break;
            case 'received':
                self.chatList.onMessageReceived(data.messageId);
                break;
            case 'read':
                self.chatList.onMessageRead(data.messageId);
                break;
        }
    });

    socket.onChannelMateUpdated(self.channel, e=>{
        if (e.payload.mate.isLocal()) return;
        //TODO update marks on change mate name
        _updateMateChoice(self);
    });

    socket.onChannelMateEntered(self.channel, e=>_updateMateChoice(self, e));
    socket.onChannelMateLeft(self.channel, e=>_updateMateChoice(self, e));
    socket.onChannelMateDisconnected(self.channel, e=>_updateMateChoice(self, e));
    socket.onChannelMateReconnected(self.channel, e=>_updateMateChoice(self, e));

    socket.onClose(e=>{ self.indicator.status = 'disconnected'; });
    socket.onError(e=>{ self.indicator.status = 'disconnected'; });

    // Set the local connection name if not initialized
    const channel = self.getChannel();
    const local = channel.getLocalMate();
    if (!(local.hasParam(self.mateNameField))) {
        let payload = {};
        payload[self.mateNameField] = _getDefaultName(self);
        channel.updateSharedData(payload);
    }
}

function _updateMateChoice(self, e) {
    let channel = self.getChannel(),
        mates = channel.getMates(),
        names = {};
    for (let id in mates) {
        let mate = mates[id];
        if (mate.isLocal()) continue;
        names[id] = mate[self.mateNameField] || 'noname';
    }
    lx(self)>>mateChoice.options(names);
}

function _onChooseMate(self, mateId) {
    lx(self)>>mateChoice.value(null);

    if (!self.chatList.has(mateId))
        self.chatList.add(mateId);

    self.chatList.focus(mateId);
}

function _getDefaultName(self) {
    let delaultNames = _getDefaultNames(),
        names = [],
        mates = self.getChannel().getMates(),
        newName = null;

    for (let i in mates) {
        let name = mates[i][self.mateNameField];
        if (name) names.push(name);
    }

    let limit = 10, attempt = 0;
    while (true) {
        let i = lx.Math.randomInteger(0, delaultNames.length - 1);
        let delaultName = delaultNames[i];
        if (names.includes(delaultName)) {
            attempt++;
            if (attempt > limit) break;
            continue;
        }
        newName = delaultName;
        break;
    }

    if (!newName) newName = lx.i18n(newMate);
    return newName;
}

function _getDefaultNames() {
    return [
        lx.i18n(Dog),
        lx.i18n(Cow),
        lx.i18n(Cat),
        lx.i18n(Horse),
        lx.i18n(Donkey),
        lx.i18n(Tiger),
        lx.i18n(Lion),
        lx.i18n(Panther),
        lx.i18n(Leopard),
        lx.i18n(Cheetah),
        lx.i18n(Bear),
        lx.i18n(Elephant),
        lx.i18n(PolarBear),
        lx.i18n(Turtle),
        lx.i18n(Tortoise),
        lx.i18n(Crocodile),
        lx.i18n(Rabbit),
        lx.i18n(Porcupine),
        lx.i18n(Hare),
        lx.i18n(Hen),
        lx.i18n(Pigeon),
        lx.i18n(Albatross),
        lx.i18n(Crow),
        lx.i18n(Fish),
        lx.i18n(Dolphin),
        lx.i18n(Frog),
        lx.i18n(Whale),
        lx.i18n(Alligator),
        lx.i18n(Eagle),
        lx.i18n(FlyingSquirrel),
        lx.i18n(Ostrich),
        lx.i18n(Fox),
        lx.i18n(Goat),
        lx.i18n(Jackal),
        lx.i18n(Emu),
        lx.i18n(Armadillo),
        lx.i18n(Eel),
        lx.i18n(Goose),
        lx.i18n(ArcticFox),
        lx.i18n(Wolf),
        lx.i18n(Beagle),
        lx.i18n(Gorilla),
        lx.i18n(Chimpanzee),
        lx.i18n(Monkey),
        lx.i18n(Beaver),
        lx.i18n(Orangutan),
        lx.i18n(Antelope),
        lx.i18n(Bat),
        lx.i18n(Badger),
        lx.i18n(Giraffe),
        lx.i18n(HermitCrab),
        lx.i18n(GiantPanda),
        lx.i18n(Hamster),
        lx.i18n(Cobra),
        lx.i18n(HammerheadShark),
        lx.i18n(Camel),
        lx.i18n(Hawk),
        lx.i18n(Deer),
        lx.i18n(Chameleon),
        lx.i18n(Hippopotamus),
        lx.i18n(Jaguar),
        lx.i18n(Chihuahua),
        lx.i18n(KingCobra),
        lx.i18n(Ibex),
        lx.i18n(Lizard),
        lx.i18n(Koala),
        lx.i18n(Kangaroo),
        lx.i18n(Iguana),
        lx.i18n(Llama),
        lx.i18n(Chinchillas),
        lx.i18n(Dodo),
        lx.i18n(Jellyfish),
        lx.i18n(Rhinoceros),
        lx.i18n(Hedgehog),
        lx.i18n(Zebra),
        lx.i18n(Possum),
        lx.i18n(Wombat),
        lx.i18n(Bison),
        lx.i18n(Bull),
        lx.i18n(Buffalo),
        lx.i18n(Sheep),
        lx.i18n(Meerkat),
        lx.i18n(Mouse),
        lx.i18n(Otter),
        lx.i18n(Sloth),
        lx.i18n(Owl),
        lx.i18n(Vulture),
        lx.i18n(Flamingo),
        lx.i18n(Racoon),
        lx.i18n(Mole),
        lx.i18n(Duck),
        lx.i18n(Swan),
        lx.i18n(Lynx),
        lx.i18n(MonitorLizard),
        lx.i18n(Elk),
        lx.i18n(Boar),
        lx.i18n(Lemur),
        lx.i18n(Mule),
        lx.i18n(Baboon),
        lx.i18n(Mammoth),
        lx.i18n(Blue),
        lx.i18n(Rat),
        lx.i18n(Snake),
        lx.i18n(Peacock)
    ];
}

function _sendMessageReceived(self, message) {
    self.getChannel().send({
        lxChatBox: self.chatId,
        type: 'received',
        messageId: message.id
    }, [message.authorId]);
}

function _sendMessageRead(self, message) {
    self.getChannel().send({
        lxChatBox: self.chatId,
        type: 'read',
        messageId: message.id
    }, [message.authorId]);
}

function _processUnread(self) {
    let unread = 0;
    for (let i in self.chatList.boxes)
        unread += self.chatList.boxes[i].unread;
    self.trigger('newUnreadMessage', self.newEvent({
        totalUnread: unread
    }));
}

function _processRead(self) {
    let unread = 0;
    for (let i in self.chatList.boxes)
        unread += self.chatList.boxes[i].unread;
    self.trigger('messageRead', self.newEvent({
        totalUnread: unread
    }));
}

class lxChatList {
    constructor(widget) {
        this.widget = widget;
        this.active = '_';
        this.boxes = {
            '_' : new lxChatBox(widget, lx(widget)>>chat.mark(0), lx.i18n(allLabel), '_')
        };
    }

    setActive(mark) {
        this.active = mark.__mateId;
    }

    isCommonActive() {
        return this.active == '_';
    }

    add(mateId) {
        if (this.has(mateId)) return;
        const channel = this.widget.getChannel();
        const mate = channel.getMate(mateId);
        if (!mate) return;
        let name = mate[this.widget.mateNameField];
        const mark = lx(this.widget)>>chat.appendMark(name);
        mark.on('beforeDropMark', e => {
            delete this.boxes[e.mark.__mateId];
        });
        this.boxes[mateId] = new lxChatBox(this.widget, mark, name, mateId);
        _initSheet(this.widget, this.getMateChatBox(mateId).getSheet());
    }

    has(mateId) {
        return (mateId in this.boxes);
    }

    focus(mateId) {
        if (!this.has(mateId)) return;
        this.boxes[mateId].focus();
        this.active = mateId;
    }

    getMateChatBox(mateId) {
        if (!this.has(mateId)) return null;
        return this.boxes[mateId];
    }

    getActiveChatBox() {
        return this.boxes[this.active];
    }

    getCommonChatBox() {
        return this.boxes['_'];
    }

    addMessage(message, messageId, senderId, isPrivate) {
        const channel = this.widget.getChannel();
        let chatBox, mate;
        if (senderId) {
            if (isPrivate) {
                this.add(senderId);
                chatBox = this.getMateChatBox(senderId);
            } else chatBox = this.getCommonChatBox();
            mate = channel.getMate(senderId);
        } else {
            chatBox = this.getActiveChatBox();
            mate = channel.getLocalMate()
        }

        return chatBox.addMessage(message, mate, messageId);
    }

    onMessageReceived(messageId) {
        for (let i in this.boxes) {
            let chatBox = this.boxes[i];
            if (chatBox.hasMessage(messageId)) {
                chatBox.onMessageReceived(messageId);
                break;
            }
        }
    }

    onMessageRead(messageId) {
        for (let i in this.boxes) {
            let chatBox = this.boxes[i];
            if (chatBox.hasMessage(messageId)) {
                chatBox.onMessageRead(messageId);
                break;
            }
        }
    }
}

class lxChatBox {
    //TODO - buffering of messages

    constructor(widget, mark, markLabel, mateId) {
        this.widget = widget;
        this.mark = mark;
        this.label = markLabel;
        this.mark.__mateId = mateId;
        this.lastAuthor = null;
        this.messages = {};
        this.messagesCount = 0;
        this.unread = 0;
    }

    getId() {
        return this.mark.__mateId;
    }

    focus() {
        lx(this.widget)>>chat.select(this.mark.index);
    }

    getSheet() {
        return lx(this.widget)>>chat.sheet(this.mark.index);
    }

    addMessage(message, mate, messageId = null) {
        let senderName = (this.lastAuthor === mate.getId())
            ? null
            : (mate.isLocal() ? 'You' : mate[this.widget.mateNameField]);
        this.lastAuthor = mate.getId();

        messageId = messageId || this.getId() + '-' + mate.getId() + '-' + Date.now();
        let processedMessage = _prepareMessage(this.widget, message, senderName);
        const messageBox = _addMessage(this.widget, this.getSheet(), processedMessage, mate.isLocal());
        const messageObj = new lxChatMessage(messageId, this, messageBox, mate);
        this.messages[messageId] = messageObj;
        this.messagesCount++;

        if (!messageBox.isDisplay()) {
            this.unread++;
            this.mark.setLabel( this.label + ' (' + this.unread + ')' );
            messageBox.displayOnce(()=>{
                this.unread--;
                this.unread
                    ? this.mark.setLabel( this.label + ' (' + this.unread + ')' )
                    : this.mark.setLabel( this.label );
                _sendMessageRead(this.widget, messageObj);
                _processRead(this.widget);
            });
            _processUnread(this.widget);
        }

        return messageObj;
    }

    hasMessage(messageId) {
        return messageId in this.messages;
    }

    onMessageReceived(messageId) {
        this.messages[messageId].setReceived();
    }

    onMessageRead(messageId) {
        this.messages[messageId].setRead();
    }
}

class lxChatMessage {
    constructor(id, chatBox, messageBox, author) {
        this.id = id;
        this.chatBox = chatBox;
        this.messageBox = messageBox;
        this.authorId = author.getId();
        this.received = false;
        this.read = false;
    }

    isVisible() {
        if (!this.messageBox) return false;
        return this.messageBox.isDisplay();
    }

    setReceived() {
        if (this.received || this.read) return;
        if (this.messageBox)
            lx(this.messageBox)>>marker.addClass('lxSocket-ChatBox-msg-received');
        this.received = true;
    }

    setRead() {
        if (this.read) return;
        if (this.messageBox) {
            if (this.received)
                lx(this.messageBox)>>marker.removeClass('lxSocket-ChatBox-msg-received');
            lx(this.messageBox)>>marker.addClass('lxSocket-ChatBox-msg-read');
        }
        this.received = true;
        this.read = true;
    }
}
// @lx:context>
