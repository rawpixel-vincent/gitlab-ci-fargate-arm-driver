package assertions

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeError struct {
	inner error
}

func (e *fakeError) Error() string {
	return fmt.Sprintf("take error: %v", e.inner)
}

func (e *fakeError) Unwrap() error {
	return e.inner
}

func (e *fakeError) Is(err error) bool {
	_, ok := err.(*fakeError)
	return ok
}

func TestErrorIs(t *testing.T) {
	newError := errors.New("new error")

	tests := map[string]struct {
		testError      error
		expectedError  error
		expectedResult bool
	}{
		"testError is equal to expected": {
			testError:      assert.AnError,
			expectedError:  assert.AnError,
			expectedResult: true,
		},
		"testError is not equal to expected": {
			testError:      newError,
			expectedError:  assert.AnError,
			expectedResult: false,
		},
		"testError is a wrapper with expected one": {
			testError:      &fakeError{inner: assert.AnError},
			expectedError:  assert.AnError,
			expectedResult: true,
		},
		"testError is a wrapper with not expected one": {
			testError:      &fakeError{inner: newError},
			expectedError:  assert.AnError,
			expectedResult: false,
		},
		"testError is a wrapper with expected one (using Is() interface)": {
			testError:      fmt.Errorf("test: %w", &fakeError{inner: newError}),
			expectedError:  new(fakeError),
			expectedResult: true,
		},
		"testError is a wrapper with not expected one (using Is() interface)": {
			testError:      fmt.Errorf("test: %w", newError),
			expectedError:  new(fakeError),
			expectedResult: false,
		},
		"testError is nil": {
			testError:      nil,
			expectedError:  assert.AnError,
			expectedResult: false,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			ttt := new(testing.T)
			result := ErrorIs(ttt, tt.testError, tt.expectedError)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
