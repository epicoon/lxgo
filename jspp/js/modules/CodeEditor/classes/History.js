#lx:public;

/*
для события ввода текста надо сохранять
	- позицию, с которой добавлялись символы
	- количество добавленных символов
событие создания нового спана - частный случай с позицией 0, количеством 1

для события удаления текста надо сохранять
	- удаленный текст
	- позицию
	- способ удаления

событие удаления спана - навешивается на следующий спан (предыдущий, если окружающие спаны слились), надо сохранить:
	- текст в удаленном спане
	- имя удаленного спана
	- позиция, если окружающие спаны слились
	- имя следующего, если окружающие спаны слились

событие вставки текста
	- узел next, перед которым все вставлено
	- offset, если последний узел вставки слился с next
	- pre если есть
	- offset для pre, если pre есть

событие удаления фрагмента текста, надо
	- текст без спанов
	- имена спанов
	- текст, вырезанный из первого граничного спана
	- текст, вырезанный из последнего граничного спана
	- событие вешается на спан next (pre если спаны слились)
	- если было слияние pre и next, то preLen
	- имя next, если спаны слились
	- порядок первого и последнего выделенных - для восстановления каретки
*/

class History {
    #lx:const
        EVENT_INPUT = 0,
        EVENT_DELETE = 1,
        EVENT_EXTRACT = 2,
        EVENT_PASTE = 3,
        EVENT_MASS_DELETE = 4;

    constructor(context) {
        this.context = context;
        this.idCounter = 0;
        this.data = [];
        this.available = 1;
        this.active = null;
        this.sleep = false;
        this.start = 0;
        this.timeout = 2000;
        __subscribe(this, context.events);
    }

    on() {
        this.available++;
    }

    off() {
        this.available--;
    }

    reset() {
        this.active = null;
        this.start = 0;
    }

    genId() {
        var id = this.idCounter;
        this.idCounter++;
        return 'e' + id;
    }

    getName(span) {
        if (span.span.getAttribute('name') === null)
            span.span.setAttribute('name', this.genId());
        return span.name();
    }

    len() {
        return this.data.length
    }

    last() {
        return this.data[ this.len() - 1 ];
    }

    addNewEvent(span, type, info, act) {
        if (type == History.EVENT_PASTE) {
            info.pre = this.getName(info.pre);
        }

        var e = new HistoryRecord(this.context, type, info, span.span.getAttribute('name'));
        if (!span.span.getAttribute('name')) span.span.setAttribute('name', e.spanId);

        // сохраняю 50 последних событий, больше вряд ли нужно
        if (this.data.length > 50) this.data.shift();
        this.data.push(e);
        if (act) {
            this.active = e;
            this.start = (new Date).getTime();
        } else this.reset();

        return e;
    }

    addEvInput(span, info) {
        if (this.checkActive(span, self::EVENT_INPUT))
            this.active.info.amt += info.amt;
        else
            this.addNewEvent(span, self::EVENT_INPUT, info, true);
    }

    addEvDelete(span, info) {
        var act = this.checkActive(span, self::EVENT_DELETE, info);

        if (!act) this.addNewEvent(span, self::EVENT_DELETE, info, true);
        else {
            var e = this.active;
            if (info.way == 0) e.info.text += info.text;
            else e.info.text = info.text + e.info.text;
        };
    }

    checkActive(span, type, info) {
        if (this.active === null) return false;
        var e = this.active,
            res = (
                e.type == type &&
                e.span().equal(span) &&
                (new Date).getTime() - this.start < this.timeout
            );
        if (type == self::EVENT_INPUT) return res;
        if (type == self::EVENT_DELETE) return (res && e.info.way == info.way);
    }

    add(type, span, info) {
        if (this.sleep) return;
        if (this.available != 1) return;
        switch (type) {
            case self::EVENT_INPUT: this.addEvInput(span, info); break;
            case self::EVENT_DELETE: this.addEvDelete(span, info); break;
            case self::EVENT_EXTRACT:
            case self::EVENT_PASTE:
            case self::EVENT_MASS_DELETE: this.addNewEvent(span, type, info); break;
        };
    }

    back() {
        if (!this.data.length) return;
        this.reset();
        var e = this.data.pop();
        e.restore();
    }
}

function __subscribe(self, eventDispatcher) {
    eventDispatcher.subscribe(EventType.HISTORY_START_FREE_OPERATION, ()=>self.sleep = true);
    eventDispatcher.subscribe(EventType.HISTORY_FINISH_FREE_OPERATION, ()=>self.sleep = false);

    const map = {};
    map[EventType.HISTORY_INPUT] = History.EVENT_INPUT;
    map[EventType.HISTORY_DELETE] = History.EVENT_DELETE;
    map[EventType.HISTORY_EXTRACT] = History.EVENT_EXTRACT;
    map[EventType.HISTORY_PASTE] = History.EVENT_PASTE;
    map[EventType.HISTORY_MASS_DELETE] = History.EVENT_MASS_DELETE;
    for (let eventName in map) {
        eventDispatcher.subscribe(
            eventName,
            data=>{
                if (data.force) self.reset();
                self.add(map[eventName], data.span, data.info);
            }
        );
    }
}
