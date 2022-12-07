// Package executors provides abstractions for executing shell scripts remotely
package executors

import (
	"context"
)

const DefaultPort = 22

// Executor is the interface to provide operations related to script execution
type Executor interface {
	// Execute connects to a host, runs the script and disconnects
	Execute(ctx context.Context, connection ConnectionSettings, script []byte) error
}

// ConnectionSettings centralizes attributes related to the remote host settings
type ConnectionSettings struct {
	Hostname   string
	Port       int
	Username   string
	PrivateKey []byte
}
