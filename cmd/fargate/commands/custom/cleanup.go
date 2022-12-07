package custom

import (
	"fmt"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/aws"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/config"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/task"
)

// NewCleanupCommand constructs the command line abstraction for the "cleanup" stage
func NewCleanupCommand() cli.Command {
	cmd := new(CleanupCommand)
	cmd.abstractCustomCommand.customCommand = cmd

	cmd.newFargate = func(logger logging.Logger, awsRegion string) aws.Fargate {
		return aws.NewFargate(logger, awsRegion)
	}
	cmd.newMetadataManager = func(logger logging.Logger, directory string) task.MetadataManager {
		return task.NewMetadataManager(logger, directory)
	}

	return cli.Command{
		Handler: cmd,
		Config: cli.Config{
			Name:    "cleanup",
			Aliases: []string{"cl"},
			Usage:   "Cleanup the environment for Custom Executor",
			Description: `
This is the implementation of Cleanup stage of the Custom Executor.

This command is deleting the AWS Fargate Task that was dedicated for
the job scripts execution.

Details about how this command is used can be found at
https://docs.gitlab.com/runner/executors/custom.html#cleanup.`,
		},
	}
}

// CleanupCommand provides data and operations related to the "cleanup" stage
type CleanupCommand struct {
	abstractCustomCommand

	cfg    config.Global
	logger logging.Logger

	awsFargate      aws.Fargate
	metadataManager task.MetadataManager

	// Wrapping constructors to make easier mocking in the unit tests
	newFargate         func(logger logging.Logger, awsRegion string) aws.Fargate
	newMetadataManager func(logger logging.Logger, directory string) task.MetadataManager
}

// CustomExecute is the "core" of the implementation for the "cleanup" stage
func (c *CleanupCommand) CustomExecute(ctx *cli.Context) error {
	err := c.init(ctx)
	if err != nil {
		return fmt.Errorf("initializing CleanupCommand: %w", err)
	}

	c.logger.Info("Executing the command")

	c.logger.Info("Fetching task data from metadata storage")
	taskData, err := c.metadataManager.Get()
	if err != nil {
		return fmt.Errorf("obtaining information about the running task: %w", err)
	}

	logger := c.logger.WithField("taskARN", taskData.TaskARN)
	logger.Info("Stopping Fargate task")
	err = c.awsFargate.StopTask(ctx.Ctx, taskData.TaskARN, c.cfg.Fargate.Cluster)
	if err != nil {
		return fmt.Errorf("stopping Fargate Task %q: %w", taskData.TaskARN, err)
	}

	logger.Info("Clear metadata related to the stopped Fargate Task")
	err = c.metadataManager.Clear()
	if err != nil {
		return fmt.Errorf("deleting metadata related to the stopped Fargate Task %q: %w", taskData.TaskARN, err)
	}

	return nil
}

func (c *CleanupCommand) init(ctx *cli.Context) error {
	c.cfg = ctx.Config()
	c.logger = ctx.Logger().
		WithField("command", "cleanup_exec")

	c.awsFargate = c.newFargate(c.logger, c.cfg.Fargate.Region)
	err := c.awsFargate.Init()
	if err != nil {
		return fmt.Errorf("initializing Fargate adapter: %w", err)
	}

	c.metadataManager = c.newMetadataManager(c.logger, c.cfg.TaskMetadata.Directory)

	return nil
}
