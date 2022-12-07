package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/assertions"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging/test"
)

func TestRunTask(t *testing.T) {
	testContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	testError := errors.New("simulated error")
	taskARN := "my-task-arn"
	logger := createTestLogger()
	testEnvVar := map[string]string{"MY_ENV_VAR": "testing"}

	tests := map[string]struct {
		initializeAdapter bool
		environmentVars   map[string]string
		platformVersion   string
		awsError          error
		expectedARN       string
		expectedError     error
	}{
		"Fargate API returning success": {
			initializeAdapter: true,
			environmentVars:   nil,
			platformVersion:   "",
			awsError:          nil,
			expectedARN:       taskARN,
			expectedError:     nil,
		},
		"Fargate API returning success when overriding env variables": {
			initializeAdapter: true,
			environmentVars:   testEnvVar,
			platformVersion:   "",
			awsError:          nil,
			expectedARN:       taskARN,
			expectedError:     nil,
		},
		"Fargate API returning success overriding platform version": {
			initializeAdapter: true,
			environmentVars:   nil,
			platformVersion:   "1.4.0",
			awsError:          nil,
			expectedARN:       taskARN,
			expectedError:     nil,
		},
		"Fargate API returning error": {
			initializeAdapter: true,
			environmentVars:   nil,
			platformVersion:   "",
			awsError:          testError,
			expectedARN:       "",
			expectedError:     testError,
		},
		"Fargate adapter not initialized error": {
			initializeAdapter: false,
			environmentVars:   nil,
			platformVersion:   "",
			awsError:          nil,
			expectedARN:       "",
			expectedError:     ErrNotInitialized,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			mockECS := new(mockEcsClient)
			defer mockECS.AssertExpectations(t)

			mockEC2 := new(mockEc2Client)
			defer mockEC2.AssertExpectations(t)

			fargate := NewFargate(logger, "us-east-1")

			connectionSettings := ConnectionSettings{
				Subnet:         "subnet-name",
				SecurityGroup:  "security-group-name",
				EnablePublicIP: true,
			}
			taskSettings := TaskSettings{
				Cluster:              "cluster-name",
				TaskDefinition:       "task-def",
				PlatformVersion:      tt.platformVersion,
				EnvironmentVariables: tt.environmentVars,
			}

			if tt.initializeAdapter {
				mockECS.On(
					"RunTaskWithContext",
					testContext,
					mock.AnythingOfType("*ecs.RunTaskInput"),
				).
					Return(
						&ecs.RunTaskOutput{
							Tasks: []*ecs.Task{{TaskArn: &taskARN}},
						}, tt.awsError,
					).
					Once()

				err := fargate.Init()
				require.NoError(t, err)

				// Overwrite initialized values with the mocks
				fargate.(*awsFargate).ecsSvc = mockECS
				fargate.(*awsFargate).ec2Svc = mockEC2
			}

			arn, err := fargate.RunTask(testContext, taskSettings, connectionSettings)

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedARN, arn, "Wrong task ARN received")
		})
	}
}

func createTestLogger() logging.Logger {
	return test.NewNullLogger()
}

func TestNewFargate(t *testing.T) {
	logger := createTestLogger()
	awsRegion := "us-east-1"
	f := NewFargate(logger, awsRegion)
	assert.NotNil(t, f, "Should instantiate the AWS client")
	assert.Equal(t, logger, f.(*awsFargate).logger, "Should have initialized the logger")
	assert.Equal(t, awsRegion, f.(*awsFargate).awsRegion, "Should have initialized the region")
}

func TestInit(t *testing.T) {
	testError := errors.New("simulated error")

	tests := map[string]struct {
		sessionCreator func(awsRegion string) (*session.Session, error)
		expectedError  error
	}{
		"Init succeeded": {
			sessionCreator: func(awsRegion string) (*session.Session, error) {
				return session.NewSession(
					&aws.Config{
						Region: aws.String(awsRegion),
					},
				)
			},
			expectedError: nil,
		},
		"Init failed due to AWS session error": {
			sessionCreator: func(_ string) (*session.Session, error) {
				return nil, testError
			},
			expectedError: testError,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			awsFargate := new(awsFargate)

			awsFargate.sessionCreator = tt.sessionCreator

			err := awsFargate.Init()

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, testError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, awsFargate.ecsSvc, "The AWS client should have been initialized")
		})
	}
}

func TestWaitUntilTaskRunning(t *testing.T) {
	testError := errors.New("simulated error")
	logger := createTestLogger()

	tests := map[string]struct {
		initializeAdapter bool
		awsError          error
		expectedError     error
	}{
		"Fargate API returning success": {
			initializeAdapter: true,
			awsError:          nil,
			expectedError:     nil,
		},
		"Fargate API returning error": {
			initializeAdapter: true,
			awsError:          testError,
			expectedError:     testError,
		},
		"Fargate adapter not initialized error": {
			initializeAdapter: false,
			awsError:          nil,
			expectedError:     ErrNotInitialized,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			mockECS := new(mockEcsClient)
			defer mockECS.AssertExpectations(t)

			mockEC2 := new(mockEc2Client)
			defer mockEC2.AssertExpectations(t)

			fargate := NewFargate(logger, "us-east-1")

			if tt.initializeAdapter {
				mockECS.On(
					"WaitUntilTasksRunningWithContext",
					mock.AnythingOfType("*context.emptyCtx"),
					mock.AnythingOfType("*ecs.DescribeTasksInput"),
				).
					Return(tt.awsError).
					Once()

				err := fargate.Init()
				require.NoError(t, err)

				// Overwrite initialized values with the mocks
				fargate.(*awsFargate).ecsSvc = mockECS
				fargate.(*awsFargate).ec2Svc = mockEC2
			}

			err := fargate.WaitUntilTaskRunning(context.Background(), "param1", "param2")

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestStopTask(t *testing.T) {
	testError := errors.New("simulated error")
	logger := createTestLogger()

	tests := map[string]struct {
		initializeAdapter bool
		awsTaskOutput     *ecs.StopTaskOutput
		awsError          error
		expectedError     error
	}{
		"Fargate API returning success": {
			initializeAdapter: true,
			awsTaskOutput:     &ecs.StopTaskOutput{},
			awsError:          nil,
			expectedError:     nil,
		},
		"Fargate API returning error and task output": {
			initializeAdapter: true,
			awsTaskOutput:     &ecs.StopTaskOutput{},
			awsError:          testError,
			expectedError:     testError,
		},
		"Fargate API returning error without task output": {
			initializeAdapter: true,
			awsTaskOutput:     nil,
			awsError:          testError,
			expectedError:     testError,
		},
		"Fargate adapter not initialized error": {
			initializeAdapter: false,
			awsTaskOutput:     nil,
			awsError:          nil,
			expectedError:     ErrNotInitialized,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			mockECS := new(mockEcsClient)
			defer mockECS.AssertExpectations(t)

			mockEC2 := new(mockEc2Client)
			defer mockEC2.AssertExpectations(t)

			fargate := NewFargate(logger, "us-east-1")

			if tt.initializeAdapter {
				mockECS.On(
					"StopTaskWithContext",
					mock.AnythingOfType("*context.emptyCtx"),
					mock.AnythingOfType("*ecs.StopTaskInput"),
				).
					Return(tt.awsTaskOutput, tt.awsError).
					Once()

				err := fargate.Init()
				require.NoError(t, err)

				// Overwrite initialized values with the mocks
				fargate.(*awsFargate).ecsSvc = mockECS
				fargate.(*awsFargate).ec2Svc = mockEC2
			}

			err := fargate.StopTask(context.Background(), "param1", "param2")

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestGetContainerIP(t *testing.T) {
	testError := errors.New("simulated error")
	logger := createTestLogger()

	tests := map[string]struct {
		initializeAdapter  bool
		usePublicIP        bool
		awsDescribeTask    error
		awsDescribeNetwork error
		expectedIP         string
		expectedError      error
	}{
		"Successfully obtained IP when private desired": {
			initializeAdapter:  true,
			usePublicIP:        false,
			awsDescribeTask:    nil,
			awsDescribeNetwork: nil,
			expectedIP:         "10.0.0.1",
			expectedError:      nil,
		},
		"Successfully obtained IP when public desired": {
			initializeAdapter:  true,
			usePublicIP:        true,
			awsDescribeTask:    nil,
			awsDescribeNetwork: nil,
			expectedIP:         "172.0.0.1",
			expectedError:      nil,
		},
		"Error during fetching task info for private IP": {
			initializeAdapter:  true,
			usePublicIP:        false,
			awsDescribeTask:    testError,
			awsDescribeNetwork: nil,
			expectedIP:         "",
			expectedError:      testError,
		},
		"Error during fetching task info for public IP": {
			initializeAdapter:  true,
			usePublicIP:        true,
			awsDescribeTask:    testError,
			awsDescribeNetwork: nil,
			expectedIP:         "",
			expectedError:      testError,
		},
		"Error during fetching ip for public IP": {
			initializeAdapter:  true,
			usePublicIP:        true,
			awsDescribeTask:    nil,
			awsDescribeNetwork: testError,
			expectedIP:         "",
			expectedError:      testError,
		},
		"Fargate adapter not initialized error": {
			initializeAdapter:  false,
			usePublicIP:        true,
			awsDescribeTask:    nil,
			awsDescribeNetwork: nil,
			expectedIP:         "",
			expectedError:      ErrNotInitialized,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			mockECS := mockFargateDescribeTask(tt.initializeAdapter, tt.awsDescribeTask)
			defer mockECS.AssertExpectations(t)

			networkAPIShouldBeCalled := tt.usePublicIP &&
				tt.awsDescribeTask == nil &&
				tt.initializeAdapter
			mockEC2 := mockFargateDescribeNetworkInterfaces(
				networkAPIShouldBeCalled, tt.awsDescribeNetwork,
			)
			defer mockEC2.AssertExpectations(t)

			fargate := NewFargate(logger, "us-east-1")
			if tt.initializeAdapter {
				err := fargate.Init()
				require.NoError(t, err)

				// Overwrite initialized values with the mocks
				fargate.(*awsFargate).ecsSvc = mockECS
				fargate.(*awsFargate).ec2Svc = mockEC2
			}

			ip, err := fargate.GetContainerIP(context.Background(), "param1", "param2", tt.usePublicIP)

			assert.Equal(t, tt.expectedIP, ip)

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func mockFargateDescribeNetworkInterfaces(shouldBeCalled bool, errorToReturn error) *mockEc2Client {
	mockEC2 := new(mockEc2Client)

	if !shouldBeCalled {
		return mockEC2
	}

	publicIP := "172.0.0.1"
	mockEC2.On(
		"DescribeNetworkInterfacesWithContext",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("*ec2.DescribeNetworkInterfacesInput"),
	).
		Return(
			&ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []*ec2.NetworkInterface{
					{
						Association: &ec2.NetworkInterfaceAssociation{
							PublicIp: &publicIP,
						},
					},
				},
			}, errorToReturn,
		).
		Once()

	return mockEC2
}

func mockFargateDescribeTask(shouldBeCalled bool, errorToReturn error) *mockEcsClient {
	mockECS := new(mockEcsClient)

	if !shouldBeCalled {
		return mockECS
	}

	privateIP := "10.0.0.1"
	expectedNetworkIDKey := "networkInterfaceId"
	expectedNetworkIDValue := "net-id"
	mockECS.On(
		"DescribeTasksWithContext",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("*ecs.DescribeTasksInput"),
	).
		Return(
			&ecs.DescribeTasksOutput{
				Tasks: []*ecs.Task{
					{
						Containers: []*ecs.Container{
							{
								NetworkInterfaces: []*ecs.NetworkInterface{
									{PrivateIpv4Address: &privateIP},
								},
							},
						},
						Attachments: []*ecs.Attachment{
							{
								Details: []*ecs.KeyValuePair{
									{
										Name:  &expectedNetworkIDKey,
										Value: &expectedNetworkIDValue,
									},
								},
							},
						},
					},
				},
			}, errorToReturn,
		).
		Once()

	return mockECS
}
