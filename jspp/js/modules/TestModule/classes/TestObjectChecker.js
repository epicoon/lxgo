#lx:namespace lx;
class TestObjectChecker {
    constructor(test, object, objectName) {
        this.test = test;
        this.object = object;
        this.objectName = objectName;
    }

    /**
     * @param name {String}
     * @param value {Any}
     * @return {lx.TestObjectChecker}
     */
    hasField(name, value= undefined) {
        if (!(name in this.object)) {
            this.test.addError('Object "' + this.objectName + '" does not have field "' + name + '"');
            return this;
        }

        if (value === undefined) return this;

        if (this.object[name] !== value)
            this.test.addError('Object`s "' + this.objectName + '" field "' + name + '" is not ' + value
                + '. The current value is: ' + this.object[name]);

        return this;
    }

    /**
     * @param structure {Object}
     * @return {lx.TestObjectChecker}
     */
    isStructure(structure) {
        __compareObjects(this, '', structure, this.object);
        return this;
    }
}

function __compareObjects(self, name, structure, object) {
    if (!lx.isObject(object) && !lx.isDomELem(object)) {
        self.test.addError('Object`s "' + self.objectName + '" field "' + name + '" has to be an object');
        return;
    }

    for (let fieldName in structure) {
        if (!(fieldName in object)) {
            self.test.addError('Object "' + self.objectName + '" does not have field "' + fieldName + '"');
            continue;
        }
        __compare(self, name + '.' + fieldName, structure[fieldName], object[fieldName]);
    }
}

function __compareArrays(self, name, structureArray, objectArray) {
    if (!lx.isArray(objectArray)) {
        self.test.addError('Object`s "' + self.objectName + '" field "' + name + '" has to be an array');
        return;
    }

    for (let i in structureArray) {
        if (i >= objectArray.length) {
            self.test.addError('Object`s "' + self.objectName + '" field "' + name
                + '" has only ' + (i - 1) + 'elements');
            return;
        }
        __compare(self, name + '.' + i, structureArray[i], objectArray[i]);
    }
}

function __compareScalars(self, name, structureValue, objectValue) {
    if (structureValue instanceof RegExp) {
        if (!lx.isString(objectValue)) {
            self.test.addError('Object`s "' + self.objectName + '" field "' + name + '" has to be a string');
            return;
        }

        if (!structureValue.test(objectValue))
            self.test.addError('Object`s "' + self.objectName + '" field "' + name
                + '" has to correspond to RegExp ' + structureValue + '. The current value is: ' + objectValue);
        return;
    }

    if (structureValue !== objectValue)
        self.test.addError('Object`s "' + self.objectName + '" field "' + name + '" is not ' + structureValue
            + '. The current value is: ' + objectValue);
}

function __compare(self, name, structureValue, objectValue) {
    if (lx.isObject(structureValue))
        __compareObjects(self, name, structureValue, objectValue);
    else if (lx.isArray(structureValue))
        __compareArrays(self, name, structureValue, objectValue)
    else
        __compareScalars(self, name, structureValue, objectValue);
}
