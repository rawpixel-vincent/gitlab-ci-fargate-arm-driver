package signal

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging/test"
)

func TestNewTerminationHandler(t *testing.T) {
	logger, output := test.NewBufferedLogger()

	th := NewTerminationHandler(logger)
	go th.HandleSignals()

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		<-th.Context().Done()

		assert.Contains(t, output.String(), "Received exit signal; quitting")

		wg.Done()
	}()

	th.stopCh <- os.Interrupt

	wg.Wait()
}
