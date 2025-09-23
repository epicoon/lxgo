#lx:public;

class LinesList {
    constructor(list) {
        this.list = list;
    }

    minIndent() {
        var res = Infinity;
        for (var i=0, l=this.list.length; i<l; i++) {
            var indentL = this.list[i].indent().length;
            if (res > indentL) res = indentL;
        }
        return res;
    }

    commented() {
        for (var i=0, l=this.list.length; i<l; i++)
            if (!this.list[i].commented()) return false;
        return true;
    }

    comment() {
        var minIndent = this.minIndent();
        for (var i=0, l=this.list.length; i<l; i++) this.list[i].comment(minIndent);
    }

    uncomment() {
        for (var i=0, l=this.list.length; i<l; i++) this.list[i].uncomment();
    }

    swapComment() {
        this.commented() ? this.uncomment() : this.comment();
    }
}
