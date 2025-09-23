// @lx:module lx.TestModule;

// @lx:require classes/;

// @lx:namespace lx;
class TestModule {
    constructor() {
        this.tests = {};
        __prepareTests(this);
    }

    /**
     * @protected
     * @return {Array}
     */
    getTestsList() {
        // pass
    }

    /**
     * @protected
     */
    beforeTest() {
        // pass
    }
    
    runTest(name) {
        this.beforeTest();
        const test = this.tests[name];
        test.run();
        return new lx.TestReport(test);
    }

    run() {
        let report = {};
        for (let name in this.tests)
            report[name] = this.runTest(name);
        return report;
    }
}

function __prepareTests(self) {
    let list = self.getTestsList();
    for (let i in list) {
        let testConstructor = list[i],
            test = new testConstructor(self);
        self.tests[test.name] = test;
    }
}
