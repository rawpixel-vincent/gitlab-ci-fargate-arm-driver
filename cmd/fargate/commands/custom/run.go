package custom

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/config"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/executors"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/executors/ssh"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/fs"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/task"
)

// ErrMissingRequiredArguments is returned when mandatory arguments are not set
var ErrMissingRequiredArguments = errors.New("missing required arguments")

// NewRunCommand constructs the command line abstraction for the "run" stage
func NewRunCommand() cli.Command {
	cmd := new(RunCommand)
	cmd.abstractCustomCommand.customCommand = cmd

	cmd.newMetadataManager = func(logger logging.Logger, directory string) task.MetadataManager {
		return task.NewMetadataManager(logger, directory)
	}
	cmd.newExecutor = func(logger logging.Logger) executors.Executor {
		return ssh.NewExecutor(logger)
	}
	cmd.newFS = func() fs.FS {
		return fs.NewOS()
	}

	return cli.Command{
		Handler: cmd,
		Config: cli.Config{
			Name:      "run",
			Aliases:   []string{"r"},
			Usage:     "Run the script from Custom Executor",
			ArgsUsage: "path_to_script stage_name",
			Description: `
This is the implementation of Run stage of the Custom Executor.

The command will execute several scripts (provided by the Runner) on the
AWS Fargate task created previously by the 'prepare' command.

The command gets two arguments:

- path_to_script - which contains the path to the script that should be executed,
- stage_name - which contains the name of the job stage that is being executed.

Details about how this command is used can be found at https://docs.gitlab.com/runner/executors/custom.html#run.`,
		},
	}
}

// RunCommand provides data and operations related to the "run" stage
type RunCommand struct {
	abstractCustomCommand

	cfg    config.Global
	logger logging.Logger

	metadataManager task.MetadataManager
	sshExecutor     executors.Executor
	fs              fs.FS

	// Wrapping constructors to make easier mocking in the unit tests
	newMetadataManager func(logger logging.Logger, directory string) task.MetadataManager
	newExecutor        func(logger logging.Logger) executors.Executor
	newFS              func() fs.FS
}

// CustomExecute is the "core" of the implementation for the "run" stage
func (c *RunCommand) CustomExecute(ctx *cli.Context) error {
	argsCnt := ctx.Cli.NArg()
	if argsCnt < 2 {
		return ErrMissingRequiredArguments
	}

	c.init(ctx)

	c.logger.Info("Executing the command")

	args := ctx.Cli.Args()
	scriptPath := args.Get(0)

	script, err := c.readFileContent(scriptPath)
	if err != nil {
		return fmt.Errorf("reading the script content: %w", err)
	}

	taskData, err := c.obtainTaskData()
	if err != nil {
		return fmt.Errorf("obtaining information about the running task: %w", err)
	}

	err = c.executeScriptOnTaskContainer(ctx.Ctx, taskData, c.cfg.SSH, script)
	if err != nil {
		return fmt.Errorf("executing the script on the remote host: %w", err)
	}

	return nil
}

func (c *RunCommand) init(ctx *cli.Context) {
	c.cfg = ctx.Config()
	c.logger = ctx.
		Logger().
		WithFields(logging.Fields{
			"command": "run_exec",
			"stage":   ctx.Cli.Args().Get(1),
		})

	c.sshExecutor = c.newExecutor(c.logger)
	c.metadataManager = c.newMetadataManager(c.logger, c.cfg.TaskMetadata.Directory)
	c.fs = c.newFS()
}

func (c *RunCommand) readFileContent(filePath string) ([]byte, error) {
	c.logger.
		WithField("file", filePath).
		Info("Reading file content")

	content, err := c.fs.ReadFile(filePath)
	if err != nil {
		return content, fmt.Errorf("reading file %q: %w", filePath, err)
	}

	return content, nil
}

func (c *RunCommand) obtainTaskData() (task.Data, error) {
	c.logger.Info("Fetching task data from metadata storage")

	fargateTaskData, err := c.metadataManager.Get()
	if err != nil {
		return fargateTaskData, fmt.Errorf("fetching data from metadata storage: %w", err)
	}

	return fargateTaskData, nil
}

func (c *RunCommand) executeScriptOnTaskContainer(ctx context.Context, taskData task.Data, sshConfig config.SSH, script []byte) error {
	c.logger.
		WithField("taskARN", taskData.TaskARN).
		Info("Executing script in the task container")

	port := sshConfig.Port
	if port < 1 {
		port = executors.DefaultPort
	}

	settings := executors.ConnectionSettings{
		Hostname:   taskData.ContainerIP,
		Port:       port,
		Username:   sshConfig.Username,
		PrivateKey: taskData.PrivateKey,
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := c.sshExecutor.Execute(runCtx, settings, script)
	if err != nil {
		return fmt.Errorf("executing script on container with IP %q: %w", taskData.ContainerIP, err)
	}

	return nil
}
