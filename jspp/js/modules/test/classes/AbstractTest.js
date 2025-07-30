#lx:namespace lx;
class AbstractTest {
    #lx:const NAME = '',
        DESCRIPTION = '';

    constructor(testModule) {
        this.testModule = testModule;
        this.errors = [];
    }

    get name() { return self::NAME; }
    get description() { return self::DESCRIPTION; }

    run() {
        // pass
    }

    /**
     * @param name {String}
     * @param object {Object}
     */
    testObject(name, object) {
        return new lx.TestObjectChecker(this, object, name);
    }

    /**
     * @return {Boolean}
     */
    hasErrors() {
        return !!this.errors.length;
    }

    /**
     * @return {Array}
     */
    getErrors() {
        return this.errors;
    }

    /**
     * @param {String} error
     */
    addError(error) {
        this.errors.push(error);
    }
}
