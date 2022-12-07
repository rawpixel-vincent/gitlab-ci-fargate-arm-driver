package cli

import (
	"github.com/urfave/cli"
	clihelpers "gitlab.com/ayufan/golang-cli-helpers"
)

type Handler interface {
	Execute(context *Context) error
}

type Config = cli.Command

type Category struct {
	Config

	SubCategories []Category
	SubCommands   []Command
}

type Command struct {
	Config

	Handler Handler
}

func (cmd Command) toActionFunc(a *App) cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		return cmd.Handler.Execute(a.makeContext(cliCtx))
	}
}

func (cmd Command) getFlags() []cli.Flag {
	return clihelpers.GetFlagsFromStruct(cmd.Handler)
}
