package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
)

type TerminationHandler struct {
	ctx      context.Context
	cancelFn func()
	logger   logging.Logger
	stopCh   chan os.Signal
}

func NewTerminationHandler(logger logging.Logger) *TerminationHandler {
	ctx, cancelFn := context.WithCancel(context.Background())

	return &TerminationHandler{
		ctx:      ctx,
		cancelFn: cancelFn,
		logger:   logger,
		stopCh:   make(chan os.Signal),
	}
}

func (th *TerminationHandler) Context() context.Context {
	return th.ctx
}

func (th *TerminationHandler) HandleSignals() {
	signal.Notify(th.stopCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-th.stopCh
	th.logger.
		WithField("signal", sig).
		Warning("Received exit signal; quitting")

	th.cancelFn()
}
