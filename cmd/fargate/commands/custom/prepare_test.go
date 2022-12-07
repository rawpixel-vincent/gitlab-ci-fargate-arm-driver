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
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/ssh"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/task"
)

type prepareCommandTestCase struct {
	context        context.Context
	fargateConfig  config.Fargate
	metadataConfig config.TaskMetadata
	taskARN        *string
	containerIP    string
	keyPair        ssh.KeyPair

	createKeyPairError      error
	fargateInitError        error
	fargateRunTaskError     error
	fargateWaitTaskError    error
	fargateContainerIPError error
	fargateStopTaskError    error
	persistARNError         error
	persistIPError          error

	shouldNotCallCreateKeyPair  bool
	shouldNotCallRunTask        bool
	shouldNotCallWaitTask       bool
	shouldNotCallGetContainerIP bool
	shouldNotCallStopTask       bool
	shouldNotPersistARN         bool
	shouldNotPersistIP          bool

	expectedError error
}

func TestNewPrepareCommand(t *testing.T) {
	cmd := NewPrepareCommand()

	assert.NotNil(t, cmd, "Command should be created")
	assert.NotNil(t, cmd.Handler, "Handler should be created")
}

func TestPrepareCommand_CustomExecute(t *testing.T) {
	testContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	testFargateConfig := config.Fargate{
		Cluster:        "cluster",
		Region:         "region",
		Subnet:         "subnet",
		SecurityGroup:  "security-group",
		TaskDefinition: "task-definition",
		EnablePublicIP: true,
	}
	testMetadataConfig := config.TaskMetadata{
		Directory: "directory",
	}
	testTaskARN := "task-arn"
	testContainerIP := "1.2.3.4"
	testKeyPair := ssh.KeyPair{
		PrivateKey: []byte("Private key"),
		PublicKey:  []byte("Public key"),
	}

	testError := errors.New("simulated error")
	testErrorStopTask := errors.New("simulated error 2")

	t.Log(testError, testErrorStopTask)

	tests := map[string]prepareCommandTestCase{
		"Error during Fargate Init": {
			fargateInitError:            testError,
			shouldNotCallCreateKeyPair:  true,
			shouldNotCallRunTask:        true,
			shouldNotCallWaitTask:       true,
			shouldNotCallGetContainerIP: true,
			shouldNotCallStopTask:       true,
			shouldNotPersistARN:         true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},
		"Error during create Public / Private Key Pair": {
			createKeyPairError:          testError,
			shouldNotCallRunTask:        true,
			shouldNotCallWaitTask:       true,
			shouldNotCallGetContainerIP: true,
			shouldNotCallStopTask:       true,
			shouldNotPersistARN:         true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},

		"Error during Fargate Run Task": {
			fargateRunTaskError:         testError,
			shouldNotCallWaitTask:       true,
			shouldNotCallGetContainerIP: true,
			shouldNotPersistARN:         true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},
		"Error during Fargate Run Task - when no taskARN was provided": {
			taskARN:                     func(s string) *string { return &s }(""),
			fargateRunTaskError:         testError,
			shouldNotCallWaitTask:       true,
			shouldNotCallGetContainerIP: true,
			shouldNotCallStopTask:       true,
			shouldNotPersistARN:         true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},
		"Error during Fargate Wait Task": {
			fargateWaitTaskError:        testError,
			shouldNotCallGetContainerIP: true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},
		"Error during Fargate Wait Task and Stop Task": {
			fargateWaitTaskError:        testError,
			fargateStopTaskError:        testErrorStopTask,
			shouldNotCallGetContainerIP: true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},
		"Error during Fargate Get Container IP": {
			fargateContainerIPError: testError,
			shouldNotPersistIP:      true,
			expectedError:           testError,
		},
		"Error during persisting task ARN": {
			persistARNError:             testError,
			shouldNotCallWaitTask:       true,
			shouldNotCallGetContainerIP: true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},
		"Error during persisting ARN and stop task": {
			persistARNError:             testError,
			fargateStopTaskError:        testErrorStopTask,
			shouldNotCallWaitTask:       true,
			shouldNotCallGetContainerIP: true,
			shouldNotPersistIP:          true,
			expectedError:               testError,
		},
		"Error during persisting container IP": {
			persistIPError: testError,
			expectedError:  testError,
		},
		"Execute prepare with success": {
			shouldNotCallStopTask: true,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			tt.context = testContext
			tt.fargateConfig = testFargateConfig
			tt.metadataConfig = testMetadataConfig
			tt.containerIP = testContainerIP
			tt.keyPair = testKeyPair
			if tt.taskARN == nil {
				tt.taskARN = &testTaskARN
			}

			mockKeyFactory := new(ssh.MockKeyFactory)
			defer mockKeyFactory.AssertExpectations(t)

			mockAwsFargate := new(aws.MockFargate)
			defer mockAwsFargate.AssertExpectations(t)

			mockMetadataManager := new(task.MockMetadataManager)
			defer mockMetadataManager.AssertExpectations(t)

			setExpectationsForKeyFactory(mockKeyFactory, tt)
			setExpectationsForFargate(mockAwsFargate, tt)
			setExpectationsForMetadataManager(mockMetadataManager, tt)

			prepare := new(PrepareCommand)
			prepare.newKeyFactory = func(logger logging.Logger) ssh.KeyFactory {
				return mockKeyFactory
			}
			prepare.newFargate = func(logger logging.Logger, awsRegion string) aws.Fargate {
				return mockAwsFargate
			}
			prepare.newMetadataManager = func(logger logging.Logger, directory string) task.MetadataManager {
				return mockMetadataManager
			}

			err := prepare.CustomExecute(createCliContextForTests(tt))

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err, "Executing the command should not return errors")
			assert.NotEmpty(t, prepare.awsFargate)
			assert.NotEmpty(t, prepare.metadataManager)
		})
	}
}

func setExpectationsForKeyFactory(mockKeyFactory *ssh.MockKeyFactory, testParams prepareCommandTestCase) {
	if testParams.shouldNotCallCreateKeyPair {
		return
	}

	mockKeyFactory.On("Create", defaultBitSize).
		Return(&testParams.keyPair, testParams.createKeyPairError).
		Once()
}

func setExpectationsForFargate(mock *aws.MockFargate, testParams prepareCommandTestCase) {
	mock.On("Init").
		Return(testParams.fargateInitError).
		Once()

	setExpectationForFargateRunTask(mock, testParams)
	setExpectationForFargateWaitTask(mock, testParams)
	setExpectationForFargateGetIP(mock, testParams)
	setExpectationForFargateStopTask(mock, testParams)
}

func setExpectationForFargateRunTask(mockAwsFargate *aws.MockFargate, testParams prepareCommandTestCase) {
	if testParams.shouldNotCallRunTask {
		return
	}

	expectedConnectionSettings := aws.ConnectionSettings{
		Subnet:         testParams.fargateConfig.Subnet,
		SecurityGroup:  testParams.fargateConfig.SecurityGroup,
		EnablePublicIP: testParams.fargateConfig.EnablePublicIP,
	}
	expectedTaskSettings := aws.TaskSettings{
		Cluster:         testParams.fargateConfig.Cluster,
		TaskDefinition:  testParams.fargateConfig.TaskDefinition,
		PlatformVersion: testParams.fargateConfig.PlatformVersion,
		EnvironmentVariables: map[string]string{
			"SSH_PUBLIC_KEY": string(testParams.keyPair.PublicKey),
		},
	}

	mockAwsFargate.On(
		"RunTask",
		testParams.context,
		expectedTaskSettings,
		expectedConnectionSettings,
	).
		Return(*testParams.taskARN, testParams.fargateRunTaskError).
		Once()
}

func setExpectationForFargateWaitTask(mockAwsFargate *aws.MockFargate, testParams prepareCommandTestCase) {
	if testParams.shouldNotCallWaitTask {
		return
	}

	mockAwsFargate.On(
		"WaitUntilTaskRunning",
		testParams.context,
		*testParams.taskARN,
		testParams.fargateConfig.Cluster,
	).
		Return(testParams.fargateWaitTaskError).
		Once()
}

func setExpectationForFargateGetIP(mockAwsFargate *aws.MockFargate, testParams prepareCommandTestCase) {
	if testParams.shouldNotCallGetContainerIP {
		return
	}

	mockAwsFargate.On("GetContainerIP",
		testParams.context,
		*testParams.taskARN,
		testParams.fargateConfig.Cluster,
		testParams.fargateConfig.EnablePublicIP,
	).
		Return(testParams.containerIP, testParams.fargateContainerIPError).
		Once()
}

func setExpectationForFargateStopTask(mockAwsFargate *aws.MockFargate, testParams prepareCommandTestCase) {
	if testParams.shouldNotCallStopTask {
		return
	}

	mockAwsFargate.On(
		"StopTask",
		testParams.context,
		*testParams.taskARN,
		testParams.fargateConfig.Cluster,
	).
		Return(testParams.fargateStopTaskError).
		Once()
}

func setExpectationsForMetadataManager(mockManager *task.MockMetadataManager, testParams prepareCommandTestCase) {
	setExpectationsForPersistingTaskARN(mockManager, testParams)
	setExpectationsForPersistingContainerIP(mockManager, testParams)
}

func setExpectationsForPersistingTaskARN(mockManager *task.MockMetadataManager, testParams prepareCommandTestCase) {
	if testParams.shouldNotPersistARN {
		return
	}

	expectedDataFirstCall := task.Data{
		TaskARN:    *testParams.taskARN,
		PrivateKey: testParams.keyPair.PrivateKey,
	}
	mockManager.On("Persist", expectedDataFirstCall).
		Return(testParams.persistARNError).
		Once()
}

func setExpectationsForPersistingContainerIP(mockManager *task.MockMetadataManager, testParams prepareCommandTestCase) {
	if testParams.shouldNotPersistIP {
		return
	}

	expectedDataSecondCall := task.Data{
		TaskARN:     *testParams.taskARN,
		ContainerIP: testParams.containerIP,
		PrivateKey:  testParams.keyPair.PrivateKey,
	}
	mockManager.On("Persist", expectedDataSecondCall).
		Return(testParams.persistIPError).
		Once()
}

func createCliContextForTests(testParams prepareCommandTestCase) *cli.Context {
	cliCtx := new(cli.Context)
	cliCtx.SetConfig(config.Global{
		Fargate:      testParams.fargateConfig,
		TaskMetadata: testParams.metadataConfig,
	})
	cliCtx.SetLogger(createTestLogger())
	cliCtx.Ctx = testParams.context

	return cliCtx
}

func createTestLogger() logging.Logger {
	return test.NewNullLogger()
}
