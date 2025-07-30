// @lx:require queues/;

// @lx:namespace lx;
class Queues extends lx.AppComponent {
    init() {
        this.list = {};
    }

    add(qName, task) {
        if (!(qName in this.list)) {
            this.list[qName] = new lx.Queue(qName);
        }

        this.list[qName].add(task);
    }

    remove(queue) {
        if (!lx.isString(queue)) queue = queue.name;
        if (queue in this.list)
            delete this.list[queue];
    }
}
