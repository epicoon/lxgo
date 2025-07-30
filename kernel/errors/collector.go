package errors

import (
	"fmt"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IError */
type Error struct {
	code uint
	text string
}

/** @constructor */
func NewError(err string) *Error {
	return &Error{text: err}
}

/** @constructor */
func NewCodifiedError(code uint, err string) *Error {
	return &Error{code: code, text: err}
}

func (err *Error) Code() uint {
	return err.code
}

func (err *Error) Error() string {
	return err.text
}

/** @interface kernel.IErrorsCollector */
type ErrorsCollector struct {
	errorsCollection []kernel.IError
}

/** @constructor */
func NewErrorsCollector() *ErrorsCollector {
	return &ErrorsCollector{errorsCollection: make([]kernel.IError, 0)}
}

func (c *ErrorsCollector) CollectError(err kernel.IError) {
	c.errorsCollection = append(c.errorsCollection, err)
}

func (c *ErrorsCollector) CollectErrorf(err string, params ...any) {
	c.CollectError(NewError(fmt.Sprintf(err, params...)))
}

func (c *ErrorsCollector) CollectCodifiedErrorf(code uint, err string, params ...any) {
	c.CollectError(NewCodifiedError(code, fmt.Sprintf(err, params...)))
}

func (c *ErrorsCollector) HasErrors() bool {
	return len(c.errorsCollection) > 0
}

func (c *ErrorsCollector) GetFirstError() kernel.IError {
	if !c.HasErrors() {
		return nil
	}
	return c.errorsCollection[0]
}
