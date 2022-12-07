package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testKey   = "test_variable_key"
	testValue = "test value"
)

func testEnvGet(t *testing.T, e Env) {
	tests := map[string]string{
		testKey:   testValue,
		"unknown": "",
	}

	for key, value := range tests {
		assert.Equal(t, value, e.Get(key))
	}
}

func TestOsEnv_Get(t *testing.T) {
	e := New()
	err := os.Setenv(testKey, testValue)
	require.NoError(t, err)

	testEnvGet(t, e)
}

func TestStubbedEnv_Get(t *testing.T) {
	e := NewWithStubs(Stubs{testKey: testValue})

	testEnvGet(t, e)
}
