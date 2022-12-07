package main

import (
	"context"
	"os"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/cmd/fargate/commands/custom"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/config"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/cli"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging/storage"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/runner"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/signal"
)

const (
	defaultConfigFile = "config.toml"
)

type globalFlags struct {
	Debug     bool   `long:"debug" description:"Set debug log level"`
	LogLevel  string `long:"log-level" description:"Set custom log level (debug, info, warning, error, fatal, panic)"`
	LogFile   string `long:"log-file" description:"File where logs should be saved"`
	LogFormat string `long:"log-format" description:"Format of log (text, json)"`

	ConfigFile string `long:"config" description:"Path to configuration file"`

	TaskDefinition  string `long:"task-def" description:"Task definition" env:"CUSTOM_ENV_FARGATE_TASK_DEFINITION"`
	PlatformVersion string `long:"platform-version" description:"Fargate platform version" env:"CUSTOM_ENV_FARGATE_PLATFORM_VERSION"`
}

var (
	global = &globalFlags{
		ConfigFile: defaultConfigFile,
	}

	closeLogFile = cli.NewNopHook()
)

func main() {
	logger := logging.New()

	ctx := startSignalHandler(logger)

	a := setUpApplication(ctx, logger)

	err := a.Run(os.Args)
	if err != nil {
		logger.
			WithError(err).
			Error("Application execution failed")

		runner.GetAdapter().GenerateExitFromError(err)
	}
}

func startSignalHandler(logger logging.Logger) context.Context {
	terminationHandler := signal.NewTerminationHandler(logger)
	go terminationHandler.HandleSignals()

	return terminationHandler.Context()
}

func setUpApplication(ctx context.Context, logger logging.Logger) *cli.App {
	a := cli.New(ctx, fargate.NAME, "Fargate driver for GitLab Runner's Custom Executor")

	a.AddBeforeFunc(func(ctx *cli.Context) error {
		ctx.SetLogger(logger)

		return nil
	})
	a.AddBeforeFunc(loadConfigurationFile)
	a.AddBeforeFunc(loadCliArgsEnvVars)
	a.AddBeforeFunc(updateLogLevel)
	a.AddBeforeFunc(updateLogFormat)
	a.AddBeforeFunc(setLoggingToFile)
	a.AddBeforeFunc(logStartupMessage)

	// If logging to file will be set, closeLogFile() will close the
	// used file. Otherwise tis a NOP call.
	a.AddAfterFunc(closeLogFile)

	a.AddGlobalFlagsFromStruct(global)

	a.RegisterCategory(custom.NewCustomCategory())

	return a
}

func logStartupMessage(ctx *cli.Context) error {
	ctx.
		Logger().
		WithFields(logging.Fields{
			"version": fargate.Version().ShortLine(),
		}).
		Infof("Starting %s", fargate.NAME)

	return nil
}

func loadConfigurationFile(ctx *cli.Context) error {
	cfg, err := config.LoadFromFile(global.ConfigFile)
	if err != nil {
		return err
	}

	ctx.SetConfig(cfg)

	return nil
}

func loadCliArgsEnvVars(ctx *cli.Context) error {
	// Override parameters from config.toml if received by command line or env variable
	config := ctx.Config()

	if global.TaskDefinition != "" {
		config.Fargate.TaskDefinition = global.TaskDefinition
	}

	if global.PlatformVersion != "" {
		config.Fargate.PlatformVersion = global.PlatformVersion
	}

	ctx.SetConfig(config)

	return nil
}

func updateLogLevel(ctx *cli.Context) error {
	logLevel := ctx.Config().LogLevel
	if global.Debug {
		logLevel = "debug"
	} else if global.LogLevel != "" {
		logLevel = global.LogLevel
	}

	if logLevel == "" {
		return nil
	}

	return ctx.Logger().SetLevel(logLevel)
}

func updateLogFormat(ctx *cli.Context) error {
	logFormat := ctx.Config().LogFormat
	if global.LogFormat != "" {
		logFormat = global.LogFormat
	}

	if logFormat == "" {
		return nil
	}

	return ctx.Logger().SetFormat(logFormat)
}

func setLoggingToFile(ctx *cli.Context) error {
	logFile := ctx.Config().LogFile
	if global.LogFile != "" {
		logFile = global.LogFile
	}

	if logFile == "" {
		return nil
	}

	logStorage := storage.NewFile(logFile)
	err := logStorage.Open()
	if err != nil {
		return err
	}

	closeLogFile = func(ctx *cli.Context) error {
		return logStorage.Close()
	}

	ctx.Logger().SetOutput(logStorage)

	return nil
}
