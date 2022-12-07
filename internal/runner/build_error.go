package runner

import (
	"fmt"
)

type BuildFailureError struct {
	msg string
	err error
}

func NewBuildFailureError(err error) *BuildFailureError {
	return &BuildFailureError{
		msg: "build failure",
		err: err,
	}
}

func (e *BuildFailureError) Format(s fmt.State, v rune) {
	_, _ = fmt.Fprintf(s, "%s: %v", e.msg, e.err)
}

func (e *BuildFailureError) Error() string {
	return e.msg
}

func (e *BuildFailureError) Is(err error) bool {
	_, ok := err.(*BuildFailureError)

	return ok
}

func (e *BuildFailureError) Unwrap() error {
	return e.err
}
