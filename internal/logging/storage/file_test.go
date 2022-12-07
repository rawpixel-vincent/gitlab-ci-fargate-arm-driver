package storage

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/assertions"
)

func TestFile_Open(t *testing.T) {
	testError := errors.New("test error")

	tests := map[string]struct {
		openFileMock  func(filename string) (*os.File, error)
		expectedError error
	}{
		"when file opening fails": {
			openFileMock:  func(filename string) (*os.File, error) { return nil, testError },
			expectedError: testError,
		},
		"when file opening works properly": {
			openFileMock:  func(filename string) (*os.File, error) { return nil, nil },
			expectedError: nil,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			oldOpenFile := openFile
			defer func() {
				openFile = oldOpenFile
			}()
			openFile = testCase.openFileMock

			s := NewFile("test-file")

			err := s.Open()
			assertions.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestFile_Close(t *testing.T) {
	testError := errors.New("test error")

	mockStorageWithCloseError := func(t *testing.T, err error) (Storage, func()) {
		s := new(MockStorage)
		cleanup := func() {
			s.AssertExpectations(t)
		}

		s.On("Close").
			Return(err).
			Once()

		return s, cleanup
	}

	tests := map[string]struct {
		mockStore     func(t *testing.T) (Storage, func())
		expectedError error
	}{
		"when storage is not set": {
			mockStore:     func(_ *testing.T) (Storage, func()) { return nil, func() {} },
			expectedError: ErrLogFileNotOpened,
		},
		"when storage Close() fails": {
			mockStore:     func(t *testing.T) (Storage, func()) { return mockStorageWithCloseError(t, testError) },
			expectedError: testError,
		},
		"when storage Close() works properly": {
			mockStore:     func(t *testing.T) (Storage, func()) { return mockStorageWithCloseError(t, nil) },
			expectedError: nil,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			fs := NewFile("test-file")

			f, ok := fs.(*File)
			require.True(t, ok)

			s, cleanup := testCase.mockStore(t)
			defer cleanup()

			f.storage = s

			err := f.Close()
			assertions.ErrorIs(t, err, testCase.expectedError)
		})
	}
}

func TestFile_Write(t *testing.T) {
	testError := errors.New("test error")

	mockStorageWithWriteError := func(t *testing.T, n int, err error) (Storage, func()) {
		s := new(MockStorage)
		cleanup := func() {
			s.AssertExpectations(t)
		}

		s.On("Write", mock.Anything).
			Return(n, err).
			Once()

		return s, cleanup
	}

	tests := map[string]struct {
		mockStore      func(t *testing.T) (Storage, func())
		expectedLength int
		expectedError  error
	}{
		"when storage is not set": {
			mockStore:      func(_ *testing.T) (Storage, func()) { return nil, func() {} },
			expectedLength: 0,
			expectedError:  ErrLogFileNotOpened,
		},
		"when storage Write() fails": {
			mockStore:      func(t *testing.T) (Storage, func()) { return mockStorageWithWriteError(t, 1, testError) },
			expectedLength: 1,
			expectedError:  testError,
		},
		"when storage Write() works properly": {
			mockStore:      func(t *testing.T) (Storage, func()) { return mockStorageWithWriteError(t, 4, nil) },
			expectedLength: 4,
			expectedError:  nil,
		},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			fs := NewFile("test-file")

			f, ok := fs.(*File)
			require.True(t, ok)

			s, cleanup := testCase.mockStore(t)
			defer cleanup()

			f.storage = s

			n, err := f.Write([]byte("test"))
			assert.Equal(t, testCase.expectedLength, n)
			assertions.ErrorIs(t, err, testCase.expectedError)
		})
	}
}
