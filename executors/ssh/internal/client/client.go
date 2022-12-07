package client

import (
	"io"

	"golang.org/x/crypto/ssh"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/executors/ssh/internal/session"
)

type Client interface {
	NewSession(stdout io.Writer, stderr io.Writer) (session.Session, error)
	Disconnect() error
}

func NewConnectClient(network string, addr string, config *ssh.ClientConfig) (Client, error) {
	c, err := ssh.Dial(network, addr, config)
	if err != nil {
		return nil, err
	}

	cli := &defaultClient{
		internal: c,
	}

	return cli, nil
}

type defaultClient struct {
	internal *ssh.Client
}

func (c *defaultClient) NewSession(stdout io.Writer, stderr io.Writer) (session.Session, error) {
	s, err := c.internal.NewSession()
	if err != nil {
		return nil, err
	}

	s.Stdout = stdout
	s.Stderr = stderr

	return session.New(s), nil
}

func (c *defaultClient) Disconnect() error {
	return c.internal.Close()
}
