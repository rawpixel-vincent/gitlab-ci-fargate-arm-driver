package custom

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/aws"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/config"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/assertions"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging/test"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/task"
)

type cleanupCommandTestCase struct {
	context       context.Context
	fargateConfig config.Fargate
	taskData      task.Data

	fargateInitError     error
	obtainTaskDataError  error
	fargateStopTaskError error
	clearMetadataError   error

	expectedError error
}

func TestNewCleanupCommand(t *testing.T) {
	cmd := NewCleanupCommand()

	assert.NotNil(t, cmd, "Command should be created")
	assert.NotNil(t, cmd.Handler, "Handler should be created")
}

func TestCustomExecuteCleanup(t *testing.T) {
	testContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	testFargateConfig := config.Fargate{
		Cluster: "cluster",
	}
	testTaskData := task.Data{
		TaskARN: "task-arn",
	}
	testError := errors.New("simulated error")

	tests := map[string]cleanupCommandTestCase{
		"Execute cleanup with success": {},
		"Error during Fargate init": {
			fargateInitError: testError,
			expectedError:    testError,
		},
		"Error reading task metadata": {
			obtainTaskDataError: testError,
			expectedError:       testError,
		},
		"Error stopping Fargate task": {
			fargateStopTaskError: testError,
			expectedError:        testError,
		},
		"Error deleting task metadata": {
			clearMetadataError: testError,
			expectedError:      testError,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			tt.context = testContext
			tt.fargateConfig = testFargateConfig
			tt.taskData = testTaskData

			mockAwsFargate := new(aws.MockFargate)
			defer mockAwsFargate.AssertExpectations(t)

			mockMetadataManager := new(task.MockMetadataManager)
			defer mockMetadataManager.AssertExpectations(t)

			mockAwsFargate.On("Init").
				Return(tt.fargateInitError).
				Once()

			// Should call get task data if initialization was successful
			shouldCallGetTaskData := tt.fargateInitError == nil
			setExpectationForGetTaskData(mockMetadataManager, shouldCallGetTaskData, tt)

			// Should call stop task if init and get task data were successful
			shouldCallStopTask := shouldCallGetTaskData && tt.obtainTaskDataError == nil
			setExpectationForStopTask(mockAwsFargate, shouldCallStopTask, tt)

			// Should call clear metadata if all process worked as expected
			shouldCallClearMetadata := shouldCallStopTask && tt.fargateStopTaskError == nil
			setExpectationForClearMetadata(mockMetadataManager, shouldCallClearMetadata, tt)

			cleanup := new(CleanupCommand)
			cleanup.newFargate = func(logger logging.Logger, awsRegion string) aws.Fargate {
				return mockAwsFargate
			}
			cleanup.newMetadataManager = func(logger logging.Logger, directory string) task.MetadataManager {
				return mockMetadataManager
			}

			err := cleanup.CustomExecute(createContextForCleanupCmdTests(tt))

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err, "Executing the command should not return errors")
			assert.NotEmpty(t, cleanup.awsFargate)
			assert.NotEmpty(t, cleanup.metadataManager)
		})
	}
}

func setExpectationForGetTaskData(mockManager *task.MockMetadataManager, shouldCall bool, testParams cleanupCommandTestCase) {
	if !shouldCall {
		return
	}

	mockManager.On("Get").
		Return(testParams.taskData, testParams.obtainTaskDataError).
		Once()
}

func setExpectationForStopTask(mockAwsFargate *aws.MockFargate, shouldCall bool, testParams cleanupCommandTestCase) {
	if !shouldCall {
		return
	}

	mockAwsFargate.On(
		"StopTask",
		testParams.context,
		testParams.taskData.TaskARN,
		testParams.fargateConfig.Cluster,
	).
		Return(testParams.fargateStopTaskError).
		Once()
}

func setExpectationForClearMetadata(mockManager *task.MockMetadataManager, shouldCall bool, testParams cleanupCommandTestCase) {
	if !shouldCall {
		return
	}

	mockManager.On("Clear").
		Return(testParams.clearMetadataError).
		Once()
}

func createContextForCleanupCmdTests(testParams cleanupCommandTestCase) *cli.Context {
	ctx := new(cli.Context)
	ctx.Ctx = testParams.context
	ctx.SetConfig(config.Global{
		Fargate:      testParams.fargateConfig,
		TaskMetadata: config.TaskMetadata{},
	})
	ctx.SetLogger(test.NewNullLogger())

	return ctx
}
