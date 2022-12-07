package encoding

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testJSONStruct struct {
	A string
	B int `json:"this_is_b"`
}

var (
	testJSONStructValue = testJSONStruct{
		A: "a",
		B: 123,
	}

	testJSONStructEncodedString = `{"A":"a","this_is_b":123}`
)

func TestJson_Decode(t *testing.T) {
	source := bytes.NewBufferString(testJSONStructEncodedString)
	var target testJSONStruct

	err := NewJSON().Decode(source, &target)
	assert.NoError(t, err)
	assert.Equal(t, testJSONStructValue, target)
}

func TestJson_Encode(t *testing.T) {
	source := testJSONStructValue
	target := new(bytes.Buffer)

	err := NewJSON().Encode(source, target)
	assert.NoError(t, err)
	assert.Equal(t, testJSONStructEncodedString, strings.TrimSpace(target.String()))
}
