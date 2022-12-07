package custom

import (
	"fmt"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/runner"
)

type customCommand interface {
	CustomExecute(ctx *cli.Context) error
}

type abstractCustomCommand struct {
	customCommand
}

func (a *abstractCustomCommand) Execute(ctx *cli.Context) error {
	err := runner.InitAdapter()
	if err != nil {
		return fmt.Errorf("couldn't initialize GitLab Runner Adapter: %w", err)
	}

	return a.CustomExecute(ctx)
}

func NewCustomCategory() cli.Category {
	return cli.Category{
		Config: cli.Config{
			Name:    "custom",
			Aliases: []string{"c"},
			Usage:   "Bindings to GitLab Runner's Custom Executor",
			Description: `These commands implement the four stages interface of the Custom Executor.

You can find more information about it at
https://docs.gitlab.com/runner/executors/custom.html#stages`,
		},
		SubCommands: []cli.Command{
			NewConfigCommand(),
			NewPrepareCommand(),
			NewRunCommand(),
			NewCleanupCommand(),
		},
	}
}
