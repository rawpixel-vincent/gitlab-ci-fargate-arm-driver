package cli

import (
	stdContext "context"

	"github.com/urfave/cli"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/config"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
)

type Context struct {
	Ctx stdContext.Context
	Cli *cli.Context

	config config.Global
	logger logging.Logger
}

func (c *Context) SetConfig(cfg config.Global) {
	c.config = cfg
}

func (c *Context) Config() config.Global {
	return c.config
}

func (c *Context) SetLogger(logger logging.Logger) {
	c.logger = logger
}

func (c *Context) Logger() logging.Logger {
	return c.logger
}
