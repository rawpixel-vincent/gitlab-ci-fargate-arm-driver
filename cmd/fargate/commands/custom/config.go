package custom

import (
	"fmt"
	"io"
	"os"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/config"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/runner"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/task"
)

// errWriteConfig will be used to wrap an internal error
type errWriteConfig struct {
	inner error
}

func (e *errWriteConfig) Error() string {
	return fmt.Sprintf("writing JSON output for the ConfigCommand: %v", e.inner)
}

func (e *errWriteConfig) Unwrap() error {
	return e.inner
}

func (e *errWriteConfig) Is(err error) bool {
	_, ok := err.(*errWriteConfig)
	return ok
}

// NewConfigCommand constructs the command line abstraction for the "config" stage
func NewConfigCommand() cli.Command {
	cmd := new(ConfigCommand)
	cmd.abstractCustomCommand.customCommand = cmd

	cmd.output = os.Stdout

	return cli.Command{
		Handler: cmd,
		Config: cli.Config{
			Name:    "config",
			Aliases: []string{"co"},
			Usage:   "Provide some configuration details for Custom Executor",
			Description: `
This is the implementation of Config stage of the Custom Executor.

This command is providing some configuration details that can be next used
by Runner's Custom Executor.

This command requires GitLab Runner 12.4 or higher to work properly.

Details about how this command is used can be found at
https://docs.gitlab.com/runner/executors/custom.html#config.`,
		},
	}
}

type writer interface {
	io.Writer
}

// ConfigCommand provides data and operations related to the "config" stage
type ConfigCommand struct {
	abstractCustomCommand

	cfg    config.Global
	logger logging.Logger
	output writer
}

// CustomExecute is the "core" of the implementation for the "config" stage
func (c *ConfigCommand) CustomExecute(ctx *cli.Context) error {
	c.init(ctx)

	c.logger.Info("Executing the command")

	hostname := c.createUniqueIdentifierForHostname()

	err := runner.GetAdapter().WriteCustomExecutorConfig(c.output, hostname)
	if err != nil {
		return &errWriteConfig{inner: err}
	}

	return nil
}

func (c *ConfigCommand) init(ctx *cli.Context) {
	c.cfg = ctx.Config()
	c.logger = ctx.Logger().
		WithField("command", "config_exec")
}

func (c *ConfigCommand) createUniqueIdentifierForHostname() string {
	runnerAdapter := runner.GetAdapter()
	runnerData := task.RunnerData{
		ShortToken: runnerAdapter.ShortToken(),
		ProjectURL: runnerAdapter.ProjectURL(),
		PipelineID: runnerAdapter.PipelineID(),
		JobID:      runnerAdapter.JobID(),
	}
	return task.GenerateFilename(runnerData)
}
