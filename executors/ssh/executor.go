package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/executors"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/executors/ssh/internal/client"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
)

// ErrNotConnected is return when a previous connection was not established
var ErrNotConnected = errors.New("not connected to server")

// errInvalidPrivateKey will be used to wrap a ssh internal error
type errInvalidPrivateKey struct {
	inner error
}

func (e *errInvalidPrivateKey) Error() string {
	return fmt.Sprintf("invalid private key: %v", e.inner)
}

func (e *errInvalidPrivateKey) Unwrap() error {
	return e.inner
}

func (e *errInvalidPrivateKey) Is(err error) bool {
	_, ok := err.(*errInvalidPrivateKey)
	return ok
}

type executor struct {
	client client.Client
	logger logging.Logger

	connectClient func(network string, addr string, config *ssh.ClientConfig) (client.Client, error)

	stdout io.Writer
	stderr io.Writer
}

// NewExecutor is the constructor for an instance of the executor interface
func NewExecutor(logger logging.Logger) executors.Executor {
	executor := new(executor)
	executor.logger = logger

	executor.connectClient = client.NewConnectClient

	executor.stdout = os.Stdout
	executor.stderr = os.Stderr

	return executor
}

func (s *executor) Execute(ctx context.Context, connection executors.ConnectionSettings, script []byte) (err error) {
	s.logger.Debug("[Execute] Will connect to server and execute the specified shell script")

	err = s.connect(connection)
	if err != nil {
		return fmt.Errorf("connecting to server: %w", err)
	}

	// Use a defer function to ensure disconnection after executing the script
	defer func() {
		disconnectErr := s.disconnect()
		if err == nil && disconnectErr != nil {
			err = fmt.Errorf("disconnecting from server: %w", disconnectErr)
		}
	}()

	err = s.executeScript(ctx, script, s.stdout, s.stderr)
	if err != nil {
		return fmt.Errorf("executing script: %w", err)
	}

	s.logger.Debug("[Execute] Successfully executed script")

	return nil
}

func (s *executor) connect(connection executors.ConnectionSettings) error {
	s.logger.Debug("[connect] Will connect to server via SSH")

	signer, err := ssh.ParsePrivateKey(connection.PrivateKey)
	if err != nil {
		return &errInvalidPrivateKey{inner: err}
	}

	config := &ssh.ClientConfig{
		User:            connection.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", connection.Hostname, connection.Port)
	cli, err := s.connectClient("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("connecting to server %q as user %q: %w", addr, connection.Username, err)
	}

	s.client = cli

	s.logger.Debug("[connect] Successfully connected to server")

	return nil
}

func (s *executor) disconnect() error {
	s.logger.Debug("[disconnect] Will disconnect from server")

	if s.client == nil {
		return nil
	}

	err := s.client.Disconnect()
	if err != nil {
		return fmt.Errorf("disconnecting from server: %w", err)
	}

	s.client = nil

	s.logger.Debug("[disconnect] Successfully disconnected from server")

	return nil
}

func (s *executor) executeScript(ctx context.Context, script []byte, stdout io.Writer, stderr io.Writer) error {
	s.logger.Debug("[executeScript] Will execute a remote script")

	if s.client == nil {
		return ErrNotConnected
	}

	session, err := s.client.NewSession(stdout, stderr)
	if err != nil {
		return fmt.Errorf("creating session for ssh client: %w", err)
	}
	defer session.Close()

	err = session.ExecuteScript(ctx, string(script))
	if err != nil {
		return fmt.Errorf("executing remote script: %w", err)
	}

	s.logger.Debug("[executeScript] Command executed")

	return nil
}
