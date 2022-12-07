package runner

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitlab-runner/executors/custom/api"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/env"
)

var (
	defaultSystemFailureExitCode = 1
	defaultBuildFailureExitCode  = 2
)

func mockEnvResolver(stubs env.Stubs) func() {
	oldEnvResolver := envResolver
	cleanup := func() {
		envResolver = oldEnvResolver
	}
	envResolver = env.NewWithStubs(withDefaultEnvStubsa(stubs))

	return cleanup
}

func withDefaultEnvStubsa(stubs env.Stubs) env.Stubs {
	envStubs := env.Stubs{
		api.SystemFailureExitCodeVariable: strconv.Itoa(defaultSystemFailureExitCode),
		api.BuildFailureExitCodeVariable:  strconv.Itoa(defaultBuildFailureExitCode),
	}

	for variable, value := range stubs {
		envStubs[variable] = value
	}

	return envStubs
}

func mockOsExiter(exiter func(int)) func() {
	oldOsExiter := osExiter
	cleanup := func() {
		osExiter = oldOsExiter
	}
	osExiter = exiter

	return cleanup
}

func TestAdapter_GenerateExitFromError(t *testing.T) {
	testError := errors.New("test-error")
	testBuildError := NewBuildFailureError(testError)

	tests := map[string]struct {
		stubs              env.Stubs
		err                error
		expectedValue      int
		expectsErrorOnLoad bool
	}{
		"build exit code variable undefined": {
			stubs:              env.Stubs{api.BuildFailureExitCodeVariable: ""},
			err:                testBuildError,
			expectedValue:      -1,
			expectsErrorOnLoad: true,
		},
		"build exit code variable invalid": {
			stubs:              env.Stubs{api.BuildFailureExitCodeVariable: "abcd"},
			err:                testBuildError,
			expectedValue:      -1,
			expectsErrorOnLoad: true,
		},
		"system exit code variable undefined": {
			stubs:              env.Stubs{api.SystemFailureExitCodeVariable: ""},
			err:                testError,
			expectedValue:      -1,
			expectsErrorOnLoad: true,
		},
		"system exit code variable invalid": {
			stubs:              env.Stubs{api.SystemFailureExitCodeVariable: "abcd"},
			err:                testError,
			expectedValue:      -1,
			expectsErrorOnLoad: true,
		},
		"build error": {
			stubs:         env.Stubs{},
			err:           testBuildError,
			expectedValue: defaultBuildFailureExitCode,
		},
		"other error": {
			stubs:         env.Stubs{},
			err:           testError,
			expectedValue: defaultSystemFailureExitCode,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			exiter := func(exitCode int) {
				assert.Equal(t, testCase.expectedValue, exitCode)
			}

			defer mockEnvResolver(testCase.stubs)()
			defer mockOsExiter(exiter)()

			guard := require.NoError
			if testCase.expectsErrorOnLoad {
				guard = require.Error
			}

			guard(t, InitAdapter())

			err := fmt.Errorf("another error layer: %w", testCase.err)
			GetAdapter().GenerateExitFromError(err)
		})
	}
}

func TestAdapter_ShortToken(t *testing.T) {
	testToken := "test-token"

	tests := map[string]struct {
		stubs         env.Stubs
		expectedValue string
	}{
		"variable is defined": {
			stubs:         env.Stubs{runnerShortTokenVariable: testToken},
			expectedValue: testToken,
		},
		"variable is not defined": {
			stubs:         env.Stubs{},
			expectedValue: unknownValue,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			defer mockEnvResolver(testCase.stubs)()

			require.NoError(t, InitAdapter())
			assert.Equal(t, testCase.expectedValue, GetAdapter().ShortToken())
		})
	}
}

func TestAdapter_ProjectURL(t *testing.T) {
	testURL := "https://gitlab.example.com/namespace/project"

	tests := map[string]struct {
		stubs         env.Stubs
		expectedValue string
	}{
		"variable is defined": {
			stubs:         env.Stubs{runnerProjectURLVariable: testURL},
			expectedValue: testURL,
		},
		"variable is not defined": {
			stubs:         env.Stubs{},
			expectedValue: unknownValue,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			defer mockEnvResolver(testCase.stubs)()

			require.NoError(t, InitAdapter())
			assert.Equal(t, testCase.expectedValue, GetAdapter().ProjectURL())
		})
	}
}

func TestAdapter_PipelineID(t *testing.T) {
	tests := map[string]struct {
		stubs              env.Stubs
		expectedValue      int64
		expectsErrorOnLoad bool
	}{
		"variable is defined": {
			stubs:              env.Stubs{runnerPipelineIDVariable: "1234"},
			expectedValue:      1234,
			expectsErrorOnLoad: false,
		},
		"variable is not defined": {
			stubs:              env.Stubs{},
			expectedValue:      0,
			expectsErrorOnLoad: false,
		},
		"variable is not an integer": {
			stubs:              env.Stubs{runnerPipelineIDVariable: "abcd"},
			expectedValue:      -1,
			expectsErrorOnLoad: true,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			defer mockEnvResolver(testCase.stubs)()

			guard := require.NoError
			if testCase.expectsErrorOnLoad {
				guard = require.Error
			}

			guard(t, InitAdapter())
			assert.Equal(t, testCase.expectedValue, GetAdapter().PipelineID())
		})
	}
}

func TestAdapter_JobID(t *testing.T) {
	tests := map[string]struct {
		stubs              env.Stubs
		expectedValue      int64
		expectsErrorOnLoad bool
	}{
		"variable is defined": {
			stubs:              env.Stubs{runnerJobIDVariable: "1234"},
			expectedValue:      1234,
			expectsErrorOnLoad: false,
		},
		"variable is not defined": {
			stubs:              env.Stubs{},
			expectedValue:      0,
			expectsErrorOnLoad: false,
		},
		"variable is not an integer": {
			stubs:              env.Stubs{runnerJobIDVariable: "abcd"},
			expectedValue:      -1,
			expectsErrorOnLoad: true,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			defer mockEnvResolver(testCase.stubs)()

			guard := require.NoError
			if testCase.expectsErrorOnLoad {
				guard = require.Error
			}

			guard(t, InitAdapter())
			assert.Equal(t, testCase.expectedValue, GetAdapter().JobID())
		})
	}
}

func TestAdapter_WriteCustomExecutorConfig(t *testing.T) {
	defer mockEnvResolver(env.Stubs{})()

	require.NoError(t, InitAdapter())

	out := new(bytes.Buffer)
	err := GetAdapter().WriteCustomExecutorConfig(out, "some-vm-name")
	assert.NoError(t, err)

	json := strings.Trim(out.String(), "\n")
	assert.Equal(t, `{"driver":{"name":"fargate","version":"dev (HEAD)"},"hostname":"some-vm-name"}`, json)
}

func TestGetAdapter_NotLoaded(t *testing.T) {
	adapter = nil
	assert.Panics(t, func() { GetAdapter() })
}
