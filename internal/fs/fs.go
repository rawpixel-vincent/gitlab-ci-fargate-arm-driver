package fs

import (
	"os"

	"github.com/spf13/afero"
)

type FS interface {
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Exists(path string) (bool, error)
	TempDir(dir string, prefix string) (string, error)
	Remove(path string) error
}

type fs struct {
	afs afero.Fs
}

func NewOS() FS {
	return &fs{afs: afero.NewOsFs()}
}

func (f *fs) ReadFile(filename string) ([]byte, error) {
	return afero.ReadFile(f.afs, filename)
}

func (f *fs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return afero.WriteFile(f.afs, filename, data, perm)
}

func (f *fs) Exists(path string) (bool, error) {
	return afero.Exists(f.afs, path)
}

func (f *fs) TempDir(dir string, prefix string) (string, error) {
	return afero.TempDir(f.afs, dir, prefix)
}

func (f *fs) Remove(path string) error {
	return f.afs.Remove(path)
}
