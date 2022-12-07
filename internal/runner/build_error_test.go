package runner

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errTest = errors.New("test error")

func TestErrBuildFailure_Error(t *testing.T) {
	err := NewBuildFailureError(errTest)
	assert.EqualError(t, err, "build failure")
}

func TestErrBuildFailure_Format(t *testing.T) {
	err := NewBuildFailureError(errTest)
	output := fmt.Sprintf("%v", err)

	assert.Equal(t, "build failure: test error", output)
}

func TestErrBuildFailure_Is(t *testing.T) {
	err := NewBuildFailureError(errTest)

	assert.True(t, err.Is(&BuildFailureError{}))
	assert.False(t, err.Is(errors.New("test")))
}

func TestErrBuildFailure_Unwrap(t *testing.T) {
	err := NewBuildFailureError(errTest)

	assert.Equal(t, errTest, err.Unwrap())
}
