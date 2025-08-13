var __queueCounter = 0;
function __getQueueKey() {
    return 'q' + __queueCounter++;
}

// @lx:namespace lx;
class Queue {
    // @lx:const TYPE_TEMPORARY = 1;
    // @lx:const TYPE_CONSTANT = 2;

    constructor(name, type = null) {
        this.name = name;
        this.type = type || lx.self(TYPE_TEMPORARY);
        this.list = {};
        this.keys = [];
        this.counter = 0;
        this.active = false;
    }

    add(task) {
        var key = __getQueueKey();
        task.__lxQKey = key;
        this.keys.push(key);
        this.list[key] = task;
        task.setQueue(this);
        if (!this.active) {
            this.active = true;
            lx.app.animation.addTimer(this);
        }
    }

    go() {
        if (this.keys.len) {
            var task = this.getTask(0);
            if (task.isPending()) return;
            if (task.isNew()) task.run();
            else if (task.isCompleted()) this.shiftTask();
        }

        if (!this.keys.len) {
            lx.app.animation.removeTimer(this);
            this.active = false;
            if (this.type == lx.self(TYPE_TEMPORARY))
                lx.app.queues.remove(this);
        }
    }

    getTask(num) {
        return this.list[this.keys[num]];
    }

    shiftTask() {
        delete this.list[this.keys[0]];
        this.keys.shift();
    }
}
