package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/assertions"
)

func newMem() FS {
	return &fs{afs: afero.NewMemMapFs()}
}

func TestNewOS(t *testing.T) {
	assert.Implements(t, (*FS)(nil), NewOS())
}

func TestFs_ReadFile(t *testing.T) {
	fs := newMem()

	data, err := fs.ReadFile("false-file")
	assert.Empty(t, data)
	assertions.ErrorIs(t, err, os.ErrNotExist)
}

func TestFs_WriteFile(t *testing.T) {
	fs := newMem()

	file := "test-file"
	content := []byte("content")

	err := fs.WriteFile(file, content, 0600)
	require.NoError(t, err)

	data, err := fs.ReadFile(file)
	assert.Equal(t, content, data)
	assert.NoError(t, err)
}

func TestFs_Exists(t *testing.T) {
	fs := newMem()

	file := "test-file"
	e, err := fs.Exists(file)
	assert.False(t, e)
	assert.NoError(t, err)

	err = fs.WriteFile(file, nil, 0600)
	require.NoError(t, err)

	e, err = fs.Exists(file)
	assert.True(t, e)
	assert.NoError(t, err)
}

func TestFs_TempDir(t *testing.T) {
	fs := newMem()

	dir := "some-dir"
	prefix := "some-prefix"

	path, err := fs.TempDir(dir, prefix)
	assert.Contains(t, path, filepath.Join(dir, prefix))
	assert.NoError(t, err)
}

func TestFs_Remove(t *testing.T) {
	fs := newMem()

	file := "test-file"
	err := fs.WriteFile(file, nil, 0600)
	require.NoError(t, err)

	e, err := fs.Exists(file)
	require.True(t, e)
	require.NoError(t, err)

	err = fs.Remove(file)
	assert.NoError(t, err)

	e, err = fs.Exists(file)
	assert.False(t, e)
	assert.NoError(t, err)
}
