package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrLogFileNotOpened = errors.New("file not opened")

type File struct {
	path    string
	storage io.WriteCloser
}

func NewFile(path string) Storage {
	return &File{
		path: path,
	}
}

var openFile = func(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
}

func (f *File) Open() error {
	var err error

	f.storage, err = openFile(f.path)
	if err != nil {
		return fmt.Errorf("couldn't open log file %q for appending: %w", f.path, err)
	}

	return nil
}

func (f *File) Close() error {
	if f.storage == nil {
		return fmt.Errorf("couldn't close log file %q: %w", f.path, ErrLogFileNotOpened)
	}

	err := f.storage.Close()
	if err != nil {
		return fmt.Errorf("couldn't close log file %q: %w", f.path, err)
	}

	return nil
}

func (f *File) Write(p []byte) (int, error) {
	if f.storage == nil {
		return 0, fmt.Errorf("couldn't write to log file %q: %w", f.path, ErrLogFileNotOpened)
	}

	return f.storage.Write(p)
}
