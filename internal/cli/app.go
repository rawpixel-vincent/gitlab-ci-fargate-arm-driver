package cli

import (
	stdContext "context"
	"fmt"

	"github.com/urfave/cli"
	clihelpers "gitlab.com/ayufan/golang-cli-helpers"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate"
)

type App struct {
	ctx *Context
	app *cli.App

	beforeFunctions Hooks
	afterFunctions  Hooks
}

func New(ctx stdContext.Context, name string, usage string) *App {
	app := cli.NewApp()
	a := &App{
		ctx:             &Context{Ctx: ctx},
		app:             app,
		beforeFunctions: make(Hooks, 0),
		afterFunctions:  make(Hooks, 0),
	}

	app.Name = name
	app.Usage = usage
	app.Author = fargate.AuthorName
	app.Email = fargate.AuthorEmail
	app.Version = fargate.Version().ShortLine()
	app.EnableBashCompletion = true

	cli.VersionPrinter = func(_ *cli.Context) {
		fmt.Print(fargate.Version().Extended())
	}

	app.Before = func(cliCtx *cli.Context) error {
		return a.beforeFunctions.Execute(a.makeContext(cliCtx))
	}
	app.After = func(cliCtx *cli.Context) error {
		return a.afterFunctions.Execute(a.makeContext(cliCtx))
	}

	return a
}

func (a *App) makeContext(cliCtx *cli.Context) *Context {
	a.ctx.Cli = cliCtx

	return a.ctx
}

func (a *App) AddGlobalFlagsFromStruct(source interface{}) {
	flags := clihelpers.GetFlagsFromStruct(source)
	a.app.Flags = append(a.app.Flags, flags...)
}

func (a *App) AddBeforeFunc(f Hook) {
	a.beforeFunctions = append(a.beforeFunctions, f)
}

func (a *App) AddAfterFunc(f Hook) {
	a.afterFunctions = append(a.afterFunctions, f)
}

func (a *App) RegisterCommand(command Command) {
	a.app.Commands = append(a.app.Commands, a.composeCommand(command))
}

func (a *App) composeCommand(command Command) cli.Command {
	config := command.Config

	config.Action = command.toActionFunc(a)
	config.Flags = command.getFlags()

	return config
}

func (a *App) RegisterCategory(category Category) {
	a.app.Commands = append(a.app.Commands, a.composeCategory(category))
}

func (a *App) composeCategory(category Category) cli.Command {
	config := category.Config

	for _, subcategory := range category.SubCategories {
		config.Subcommands = append(config.Subcommands, a.composeCategory(subcategory))
	}

	for _, subcommand := range category.SubCommands {
		config.Subcommands = append(config.Subcommands, a.composeCommand(subcommand))
	}

	return config
}

func (a *App) Run(args []string) error {
	return a.app.Run(args)
}
