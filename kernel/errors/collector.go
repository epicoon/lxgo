// Package errors provides the default kernel.IError implementation (Error)
// and a kernel.IErrorsCollector (ErrorsCollector) for accumulating them -
// e.g. while validating a kernel.IForm.
package errors

import (
	"fmt"

	"github.com/epicoon/lxgo/kernel"
)

/** @interface kernel.IError */

// Error is the default kernel.IError implementation - a message with an
// optional numeric code (0 if unset).
type Error struct {
	code uint
	text string
}

var _ kernel.IError = (*Error)(nil)

/** @constructor */

// NewError constructs an Error with code 0.
func NewError(err string) *Error {
	return &Error{text: err}
}

/** @constructor */

// NewCodifiedError constructs an Error with the given code.
func NewCodifiedError(code uint, err string) *Error {
	return &Error{code: code, text: err}
}

// Code returns the error's numeric code.
func (err *Error) Code() uint {
	return err.code
}

// Error returns the error message.
func (err *Error) Error() string {
	return err.text
}

/** @interface kernel.IErrorsCollector */

// ErrorsCollector is the default kernel.IErrorsCollector implementation.
type ErrorsCollector struct {
	errorsCollection []kernel.IError
}

var _ kernel.IErrorsCollector = (*ErrorsCollector)(nil)

/** @constructor */

// NewErrorsCollector constructs an empty ErrorsCollector.
func NewErrorsCollector() *ErrorsCollector {
	return &ErrorsCollector{errorsCollection: make([]kernel.IError, 0)}
}

// CollectError adds err to the collection.
func (c *ErrorsCollector) CollectError(err kernel.IError) {
	c.errorsCollection = append(c.errorsCollection, err)
}

// CollectErrorf adds a formatted, uncoded error to the collection.
func (c *ErrorsCollector) CollectErrorf(err string, params ...any) {
	c.CollectError(NewError(fmt.Sprintf(err, params...)))
}

// CollectCodifiedErrorf adds a formatted error with the given code to the collection.
func (c *ErrorsCollector) CollectCodifiedErrorf(code uint, err string, params ...any) {
	c.CollectError(NewCodifiedError(code, fmt.Sprintf(err, params...)))
}

// HasErrors reports whether any errors were collected.
func (c *ErrorsCollector) HasErrors() bool {
	return len(c.errorsCollection) > 0
}

// GetFirstError returns the first collected error, or nil.
func (c *ErrorsCollector) GetFirstError() kernel.IError {
	if !c.HasErrors() {
		return nil
	}
	return c.errorsCollection[0]
}
