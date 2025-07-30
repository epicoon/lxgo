#lx:namespace lx;
class TestReport {
    constructor(test) {
        this.testName = test.name;
        this.testDescription = test.description;
        this.isSuccessfull = !test.hasErrors();
        this.errors = test.getErrors();
    }
}
