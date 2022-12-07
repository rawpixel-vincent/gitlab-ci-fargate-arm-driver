// Package aws provides abstractions that will be used for managing Amazon AWS resources
package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
)

var (
	defaultContainerName = "ci-coordinator"

	// ErrNotInitialized is returned when the fargate methods are invoked without initialization
	ErrNotInitialized = errors.New("fargate adapter is not initialized")
)

// Fargate should be used to manage AWS Fargate Tasks (start, stop, etc)
type Fargate interface {
	// RunTask starts a new task in a pre configured Fargate
	RunTask(ctx context.Context, taskSettings TaskSettings, connection ConnectionSettings) (string, error)

	// WaitUntilTaskRunning blocks the request until the task is in "running"
	// state or failed to reach this state
	WaitUntilTaskRunning(ctx context.Context, taskARN string, cluster string) error

	// RunTask stops a specified Fargate task
	StopTask(ctx context.Context, taskARN string, cluster string) error

	// GetContainerIP returns the IP of the container related to the specified task
	GetContainerIP(ctx context.Context, taskARN string, cluster string, usePublicIP bool) (string, error)

	// Init initialize variables and executes necessary procedures
	Init() error
}

// TaskSettings centralizes attributes related to the task configuration
type TaskSettings struct {
	Cluster              string
	TaskDefinition       string
	PlatformVersion      string
	EnvironmentVariables map[string]string
}

// ConnectionSettings centralizes attributes related to the task's network configuration
type ConnectionSettings struct {
	Subnet         string
	SecurityGroup  string
	EnablePublicIP bool
}

type ecsClient interface {
	RunTaskWithContext(aws.Context, *ecs.RunTaskInput, ...request.Option) (*ecs.RunTaskOutput, error)
	WaitUntilTasksRunningWithContext(aws.Context, *ecs.DescribeTasksInput, ...request.WaiterOption) error
	StopTaskWithContext(aws.Context, *ecs.StopTaskInput, ...request.Option) (*ecs.StopTaskOutput, error)
	DescribeTasksWithContext(aws.Context, *ecs.DescribeTasksInput, ...request.Option) (*ecs.DescribeTasksOutput, error)
}

type ec2Client interface {
	DescribeNetworkInterfacesWithContext(aws.Context, *ec2.DescribeNetworkInterfacesInput, ...request.Option) (*ec2.DescribeNetworkInterfacesOutput, error)
}

type awsFargate struct {
	logger    logging.Logger
	awsRegion string
	ecsSvc    ecsClient
	ec2Svc    ec2Client

	// The AWS NewSession function was encapsulated into the sessionCreator
	// to make easier creating unit tests
	sessionCreator func(awsRegion string) (*session.Session, error)
}

// NewFargate is a constructor for the concrete type of the Fargate interface
func NewFargate(logger logging.Logger, awsRegion string) Fargate {
	awsFargate := new(awsFargate)

	awsFargate.logger = logger
	awsFargate.awsRegion = awsRegion
	awsFargate.sessionCreator = func(awsRegion string) (*session.Session, error) {
		return session.NewSession(
			&aws.Config{
				Region: aws.String(awsRegion),
			},
		)
	}

	return awsFargate
}

func (a *awsFargate) Init() error {
	sess, err := a.sessionCreator(a.awsRegion)
	if err != nil {
		return fmt.Errorf("couldn't create AWS session: %w", err)
	}

	a.ecsSvc = ecs.New(sess)
	a.ec2Svc = ec2.New(sess)

	return nil
}

func (a *awsFargate) RunTask(ctx context.Context, taskSettings TaskSettings, connection ConnectionSettings) (string, error) {
	err := a.errIfNotInitialized()
	if err != nil {
		return "", fmt.Errorf("could not start AWS Fargate Task: %w", err)
	}

	a.logger.Debug("[RunTask] Will start a new Fargate Task")

	publicIP := "DISABLED"
	if connection.EnablePublicIP {
		publicIP = "ENABLED"
	}

	var platformVersion *string
	if taskSettings.PlatformVersion != "" {
		platformVersion = &taskSettings.PlatformVersion
	}

	taskInput := ecs.RunTaskInput{
		TaskDefinition: &taskSettings.TaskDefinition,
		Cluster:        &taskSettings.Cluster,
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				SecurityGroups: []*string{&connection.SecurityGroup},
				Subnets:        []*string{&connection.Subnet},
				AssignPublicIp: &publicIP,
			},
		},
		Overrides:       a.processEnvVariablesToInject(taskSettings.EnvironmentVariables),
		PlatformVersion: platformVersion,
	}

	taskOutput, err := a.ecsSvc.RunTaskWithContext(ctx, &taskInput)
	if err != nil {
		return "", fmt.Errorf("error starting AWS Fargate Task: %w", err)
	}

	taskARN := *taskOutput.Tasks[0].TaskArn

	a.logger.
		WithField("task-arn", taskARN).
		Debug("[RunTask] Fargate Task started with success")

	return taskARN, nil
}

func (a *awsFargate) errIfNotInitialized() error {
	if a.ecsSvc != nil && a.ec2Svc != nil {
		return nil
	}

	return ErrNotInitialized
}

func (a *awsFargate) processEnvVariablesToInject(envVars map[string]string) *ecs.TaskOverride {
	if (envVars == nil) || (len(envVars) == 0) {
		return nil
	}

	environmentVars := make([]*ecs.KeyValuePair, 0)
	for key, value := range envVars {
		environmentVars = append(environmentVars, &ecs.KeyValuePair{
			Name:  &key,
			Value: &value,
		})
	}

	taskOverride := &ecs.TaskOverride{
		ContainerOverrides: []*ecs.ContainerOverride{
			{
				Name:        &defaultContainerName,
				Environment: environmentVars,
			},
		},
	}

	a.logger.Debug("[processEnvVariablesToInject] Environment variables processed with success")

	return taskOverride
}

func (a *awsFargate) WaitUntilTaskRunning(ctx context.Context, taskARN string, cluster string) error {
	err := a.errIfNotInitialized()
	if err != nil {
		return fmt.Errorf("could not wait AWS Fargate task: %w", err)
	}

	a.logger.
		WithField("task-arn", taskARN).
		Debug(`[WaitUntilTaskRunning] Will wait until Fargate task is in "Running" state`)

	input := ecs.DescribeTasksInput{
		Cluster: &cluster,
		Tasks:   []*string{&taskARN},
	}

	err = a.ecsSvc.WaitUntilTasksRunningWithContext(ctx, &input)
	if err != nil {
		return fmt.Errorf(`error waiting AWS Fargate Task %q to be in "Running" state: %w`, taskARN, err)
	}

	a.logger.
		WithField("task-arn", taskARN).
		Debug(`[WaitUntilTaskRunning] Fargate Task in "Running" state`)

	return nil
}

func (a *awsFargate) StopTask(ctx context.Context, taskARN string, cluster string) error {
	err := a.errIfNotInitialized()
	if err != nil {
		return fmt.Errorf("could not stop AWS Fargate Task: %w", err)
	}

	a.logger.
		WithField("task-arn", taskARN).
		Debug("[StopTask] Will stop the task")

	input := ecs.StopTaskInput{
		Cluster: &cluster,
		Task:    &taskARN,
	}

	_, err = a.ecsSvc.StopTaskWithContext(ctx, &input)
	if err != nil {
		return fmt.Errorf("error stopping AWS Fargate Task %q: %w", taskARN, err)
	}

	a.logger.
		WithField("task-arn", taskARN).
		Debug("[StopTask] Fargate Task stopped with success")

	return nil
}

func (a *awsFargate) GetContainerIP(ctx context.Context, taskARN string, cluster string, usePublicIP bool) (string, error) {
	err := a.errIfNotInitialized()
	if err != nil {
		return "", fmt.Errorf("could not get container IP: %w", err)
	}

	a.logger.
		WithField("task-arn", taskARN).
		Debug("[GetContainerIP] Will get the IP for the task")

	taskDetails, err := a.getTaskDetails(ctx, taskARN, cluster)
	if err != nil {
		return "", fmt.Errorf("error accessing information about the task %q: %w", taskARN, err)
	}

	if !usePublicIP {
		privateIP := a.extractPrivateIP(taskDetails)
		return privateIP, nil
	}

	networkInterfaceID := a.extractNetworkInterfaceID(taskDetails)
	publicIP, err := a.getPublicIP(ctx, networkInterfaceID)
	if err != nil {
		return "", fmt.Errorf("error trying to get the public IP: %w", err)
	}

	a.logger.Debug("[GetContainerIP] Ip fetched with success")

	return publicIP, nil
}

func (a *awsFargate) getTaskDetails(ctx context.Context, taskARN string, cluster string) (*ecs.DescribeTasksOutput, error) {
	dt, err := a.ecsSvc.DescribeTasksWithContext(
		ctx,
		&ecs.DescribeTasksInput{
			Cluster: &cluster,
			Tasks:   []*string{&taskARN},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error during call to describe fargate task: %w", err)
	}

	a.logger.Debug("[getTaskDetails] Finished fetching the task details")

	return dt, nil
}

func (a *awsFargate) extractPrivateIP(descOut *ecs.DescribeTasksOutput) string {
	privateIP := *descOut.
		Tasks[0].
		Containers[0].
		NetworkInterfaces[0].
		PrivateIpv4Address

	a.logger.
		WithField("private-ip", privateIP).
		Debug("[extractPrivateIP] Finished extracting the private IP")

	return privateIP
}

func (a *awsFargate) extractNetworkInterfaceID(descOut *ecs.DescribeTasksOutput) string {
	var networkInterfaceID string

	for _, detail := range descOut.Tasks[0].Attachments[0].Details {
		if *detail.Name == "networkInterfaceId" {
			networkInterfaceID = *detail.Value
			break
		}
	}

	a.logger.
		WithField("network-interface-id", networkInterfaceID).
		Debug("[extractNetworkInterfaceID] Finished extracting the network interface ID")

	return networkInterfaceID
}

func (a *awsFargate) getPublicIP(ctx context.Context, networkInterfaceID string) (string, error) {
	dni, err := a.ec2Svc.DescribeNetworkInterfacesWithContext(
		ctx,
		&ec2.DescribeNetworkInterfacesInput{
			NetworkInterfaceIds: []*string{&networkInterfaceID},
		},
	)
	if err != nil {
		return "", fmt.Errorf("error reading network interfaces: %w", err)
	}

	publicIP := *dni.NetworkInterfaces[0].Association.PublicIp
	a.logger.
		WithField("public-ip", publicIP).
		Debug("[getPublicIP] Finished fetching the public IP")

	return publicIP, nil
}
