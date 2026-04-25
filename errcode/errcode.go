package errcode

import "fmt"

type Error struct {
	Code    int
	Message string
}

func New(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

func (e *Error) Error() string {
	return fmt.Sprintf("errcode %d: %s", e.Code, e.Message)
}

func IsErrCode(err error) (*Error, bool) {
	if e, ok := err.(*Error); ok {
		return e, true
	}
	return nil, false
}
