package custom

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/assertions"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging/test"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/runner"
)

func TestNewConfigCommand(t *testing.T) {
	cmd := NewConfigCommand()

	assert.NotNil(t, cmd, "Command should be created")
	assert.NotNil(t, cmd.Handler, "Handler should be created")
}

func TestCustomExecuteConfig(t *testing.T) {
	initializeAdapterForTesting(t)

	testErrorAdapterInternal := new(errWriteConfig)

	outputWriteMatcher := func(t *testing.T) interface{} {
		return mock.MatchedBy(func(data []byte) bool {
			var info struct {
				Driver struct {
					Name string `json:"name"`
				}
				Hostname string `json:"hostname"`
			}

			err := json.Unmarshal(data, &info)
			if !assert.NoError(t, err) {
				return false
			}

			return assert.Equal(t, fargate.NAME, info.Driver.Name) &&
				assert.Equal(t, "token-96688db3e5e6c32b47fe3d59f4384354c5509e27b0d48742809a8c35cca0e88d", info.Hostname)
		})
	}

	tests := map[string]struct {
		assertOutput  func(t *testing.T, output *mockWriter)
		expectedError error
	}{
		"Execute config with success": {
			assertOutput: func(t *testing.T, output *mockWriter) {
				output.On("Write", outputWriteMatcher(t)).
					Return(0, nil).
					Once()
			},
			expectedError: nil,
		},
		"Error on write config": {
			assertOutput: func(t *testing.T, output *mockWriter) {
				output.On("Write", outputWriteMatcher(t)).
					Return(0, assert.AnError).
					Once()
			},
			expectedError: testErrorAdapterInternal,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			outputMock := new(mockWriter)
			defer outputMock.AssertExpectations(t)

			tt.assertOutput(t, outputMock)

			config := new(ConfigCommand)
			config.output = outputMock

			err := config.CustomExecute(createContextForConfigCmdTests())

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err, "Executing the command should not return errors")
		})
	}
}

func initializeAdapterForTesting(t *testing.T) {
	err := os.Setenv("BUILD_FAILURE_EXIT_CODE", "1")
	require.NoError(t, err)

	err = os.Setenv("SYSTEM_FAILURE_EXIT_CODE", "1")
	require.NoError(t, err)

	err = os.Setenv("CUSTOM_ENV_CI_RUNNER_SHORT_TOKEN", "token")
	require.NoError(t, err)

	err = os.Setenv("CUSTOM_ENV_CI_PROJECT_URL", "project-URL")
	require.NoError(t, err)

	err = os.Setenv("CUSTOM_ENV_CI_PIPELINE_ID", "1")
	require.NoError(t, err)

	err = os.Setenv("CUSTOM_ENV_CI_JOB_ID", "1")
	require.NoError(t, err)

	err = runner.InitAdapter()
	require.NoError(t, err)
}

func createContextForConfigCmdTests() *cli.Context {
	ctx := new(cli.Context)
	ctx.SetLogger(test.NewNullLogger())
	return ctx
}
