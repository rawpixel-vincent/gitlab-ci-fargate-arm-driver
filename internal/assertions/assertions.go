package assertions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ErrorIs(t *testing.T, err error, expected error) bool {
	return assert.Truef(
		t,
		errors.Is(err, expected),
		"Unexpected error: %#v is not %#v",
		err,
		expected,
	)
}
