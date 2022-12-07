package storage

import (
	"io"
)

type Storage interface {
	io.WriteCloser

	Open() error
}
